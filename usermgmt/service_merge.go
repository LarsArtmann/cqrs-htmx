package usermgmt

import (
	"context"
	"slices"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// mergeChainHopLimit bounds the walk along MergedInto pointers when the
// service decides whether a merged-away loser already belongs to the
// requested survivor. Deciders cannot build a cycle (a merged survivor is
// refused), so the limit is defence against a corrupt journal, matching the
// read model's own chain limit.
const mergeChainHopLimit = 64

// MergeUsers folds every loser identity into the survivor and retires the
// losers. The merge is resumable: every step is idempotent, so a call that
// fails halfway can simply be made again — already-adopted assets and
// already-retired losers are recognised and skipped.
//
// The steps, in order:
//
//  1. validation (typed rejections for malformed input),
//  2. one MergeUsers command on the survivor that atomically adopts the
//     losers' credentials, external accounts, TOTP secret and verified-email
//     proof, then records a single UserMerged event,
//  3. per loser: membership move (role-set union, survivor first, loser
//     removal after), bot transfer, session revocation,
//  4. one MergeUserInto tombstone per loser, LAST — until it lands the loser
//     is still live and a failed merge is retried, never left half-dead.
//
// A loser already merged into this survivor (directly or through a chain) is
// skipped, which is what makes re-runs idempotent. A loser merged elsewhere
// answers a Conflict naming the survivor it points at; a deleted loser
// answers a Conflict (terminal state), a survivor that was itself merged
// away or deleted is refused before anything is dispatched.
func (s *Service) MergeUsers(ctx context.Context, survivorID UserID, loserIDs []UserID, reason string) error {
	if err := validateMergeUsersInput(survivorID, loserIDs); err != nil {
		return err
	}

	survivorAggID, err := aggIDFromUser(survivorID)
	if err != nil {
		return err
	}
	survivorState, _, err := s.repository.LoadRef(ctx, id.NewStreamRef(aggregateTypeUser, survivorAggID))
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.merge_users.load_failed", "load survivor state")
	}
	if err := requireLiveSurvivor(survivorState); err != nil {
		return err
	}

	contributions, liveLosers, err := s.collectMergeContributions(ctx, survivorID, loserIDs)
	if err != nil {
		return err
	}
	if len(liveLosers) == 0 {
		return nil
	}

	mergeCmd := identitymodel.NewMergeUsersCmd(survivorAggID, contributions, reason)
	if err := s.dispatcher.Dispatch(ctx, mergeCmd); err != nil {
		return classifyDispatchError(err, survivorID, "reason", reason)
	}

	for _, loserID := range liveLosers {
		if err := s.moveMembershipsToSurvivor(ctx, survivorID, loserID); err != nil {
			return err
		}
		if err := s.BotService.transferBotsToSurvivor(ctx, survivorID, loserID, reason); err != nil {
			return err
		}
		s.revokeSessionsBestEffort(ctx, loserID, "failed to revoke sessions on merge")
	}

	return s.retireMergeLosers(ctx, survivorID, liveLosers, reason)
}

// retireMergeLosers tombstones every live loser — the LAST step of a merge
// (D1): until it lands the loser is still live, so a failed merge is
// retried rather than left half-dead.
func (s *Service) retireMergeLosers(
	ctx context.Context, survivorID UserID, liveLosers []UserID, reason string,
) error {
	for _, loserID := range liveLosers {
		loserAggID, err := aggIDFromUser(loserID)
		if err != nil {
			return err
		}
		tombstone := identitymodel.NewMergeUserIntoCmd(loserAggID, survivorID, reason)
		if err := s.dispatcher.Dispatch(ctx, tombstone); err != nil {
			return classifyDispatchError(err, loserID, "survivor_id", survivorID.Get().String())
		}
	}
	return nil
}

// validateMergeUsersInput refuses malformed merge requests before anything
// is loaded or dispatched (D14: rejections are reserved for malformed input).
func validateMergeUsersInput(survivorID UserID, loserIDs []UserID) error {
	if survivorID.IsZero() {
		return errorfamily.NewRejection("usermgmt.merge_users.survivor_required", "survivor ID is required")
	}
	if len(loserIDs) == 0 {
		return errorfamily.NewRejection("usermgmt.merge_users.no_losers", "a merge needs at least one loser")
	}
	for i, loser := range loserIDs {
		if loser.Get().String() == survivorID.Get().String() {
			return errorfamily.NewRejection(
				"usermgmt.merge_users.self_merge", "a user cannot be merged into itself",
			).WithContextAny("loser_id", loser.Get().String())
		}
		if slices.ContainsFunc(loserIDs[i+1:], func(other UserID) bool {
			return other.Get().String() == loser.Get().String()
		}) {
			return errorfamily.NewRejection(
				"usermgmt.merge_users.duplicate_loser", "the loser list names the same user twice",
			).WithContextAny("loser_id", loser.Get().String())
		}
	}
	return nil
}

// collectMergeContributions loads every loser's fold state, answers the typed
// rejections and conflicts the deciders would answer (so a malformed merge
// fails before any event lands), skips losers already absorbed by this
// survivor, and builds the contribution list from the fold state. The fold is
// the authoritative projection of the stream: unlike a read model it cannot
// lag behind the journal the deciders decide against.
func (s *Service) collectMergeContributions(
	ctx context.Context,
	survivorID UserID,
	loserIDs []UserID,
) ([]identitymodel.MergeContribution, []UserID, error) {
	contributions := make([]identitymodel.MergeContribution, 0, len(loserIDs))
	liveLosers := make([]UserID, 0, len(loserIDs))
	for _, loserID := range loserIDs {
		loserAggID, err := aggIDFromUser(loserID)
		if err != nil {
			return nil, nil, err
		}
		state, _, err := s.repository.LoadRef(ctx, id.NewStreamRef(aggregateTypeUser, loserAggID))
		if err != nil {
			return nil, nil, errorfamily.WrapInfrastructure(
				err, "usermgmt.merge_users.load_failed", "load loser state",
			).WithContextAny("loser_id", loserID.Get().String())
		}
		if state.IsMerged() {
			if s.mergeChainResolvesTo(ctx, state.MergedInto, survivorID) {
				continue
			}
			return nil, nil, errorfamily.NewConflict(
				"usermgmt.merge_users.loser_merged",
				"loser was already merged into another user",
			).
				WithContext("merged_into", state.MergedInto.Get().String()).
				WithContextAny("loser_id", loserID.Get().String())
		}
		if !state.Exists() {
			return nil, nil, errorfamily.NewRejection("usermgmt.merge_users.loser_not_found", "loser was never registered").
				WithContextAny("loser_id", loserID.Get().String())
		}
		if state.Deleted {
			return nil, nil, errorfamily.NewConflict("usermgmt.merge_user_into.deleted", "only a live user can be merged away").
				WithContextAny("loser_id", loserID.Get().String())
		}
		if err := s.refuseThirdPartyExternalAccounts(survivorID, loserID, state.ExternalAccounts); err != nil {
			return nil, nil, err
		}

		contribution := identitymodel.MergeContribution{
			UserID:           loserID,
			Email:            state.Email,
			EmailVerified:    state.EmailVerified,
			Credentials:      state.Credentials,
			ExternalAccounts: state.ExternalAccounts,
			TOTPSecret:       nil, // filled below, only while TOTP is enabled
		}
		if state.TOTPEnabled {
			contribution.TOTPSecret = state.TOTPSecret
		}
		contributions = append(contributions, contribution)
		liveLosers = append(liveLosers, loserID)
	}
	return contributions, liveLosers, nil
}

// refuseThirdPartyExternalAccounts refuses a merge whose adoption would move
// an external account that another live user holds: the survivor's
// ExternalAccountLinked event would steal the read-model index entry from
// that user. Both rows are named so the admin can unlink the right one.
func (s *Service) refuseThirdPartyExternalAccounts(
	survivorID, loserID UserID,
	accounts []identitymodel.ExternalAccount,
) error {
	for _, account := range accounts {
		holder, ok := s.readModel.FindByExternalAccount(account.Provider, account.Subject)
		if !ok || holder.ID.Get().String() == survivorID.Get().String() ||
			holder.ID.Get().String() == loserID.Get().String() {
			continue
		}
		return errorfamily.NewConflict(
			"usermgmt.merge_users.external_account_conflict",
			"external account is linked to another live user; unlink it there first",
		).
			WithContextAny("provider", account.Provider).
			WithContextAny("subject", account.Subject).
			WithContextAny("loser_id", loserID.Get().String()).
			WithContextAny("holder_id", holder.ID.Get().String())
	}
	return nil
}

// mergeChainResolvesTo walks the MergedInto pointers from a merged-away user
// and reports whether the chain ends at the target. The walk is bounded by
// mergeChainHopLimit; a cycle answers false, like the read-model resolver.
func (s *Service) mergeChainResolvesTo(ctx context.Context, from, target UserID) bool {
	current := from
	for range mergeChainHopLimit {
		if current.IsZero() {
			return false
		}
		if current.Get().String() == target.Get().String() {
			return true
		}
		aggID, err := aggIDFromUser(current)
		if err != nil {
			return false
		}
		state, _, err := s.repository.LoadRef(ctx, id.NewStreamRef(aggregateTypeUser, aggID))
		if err != nil || !state.IsMerged() {
			return false
		}
		current = state.MergedInto
	}
	return false
}

// moveMembershipsToSurvivor gives the survivor every membership the loser
// holds: the UNION of both role sets per tenant (no role is ever lost), the
// survivor's add or update first, the loser's removal after. Re-runs are
// no-ops: the union of equal sets dispatches nothing and an already-removed
// membership is no longer listed.
func (s *Service) moveMembershipsToSurvivor(ctx context.Context, survivorID, loserID UserID) error {
	survivorActor := ActorIDFromUser(survivorID)
	survivorRoles := make(map[TenantID][]Role)
	for _, mem := range s.membershipReadModel.FindByActor(survivorID.Get().String()) {
		survivorRoles[mem.TenantID] = mem.Roles
	}
	for _, mem := range s.membershipReadModel.FindByActor(loserID.Get().String()) {
		merged := unionRoles(survivorRoles[mem.TenantID], mem.Roles)
		if _, ok := survivorRoles[mem.TenantID]; ok {
			update := NewUpdateMemberRolesCmd(survivorActor, mem.TenantID, merged)
			if err := s.dispatcher.Dispatch(ctx, update); err != nil {
				return classifyDispatchError(err, survivorID,
					"tenant_id", mem.TenantID.Get(),
					"loser_id", loserID.Get().String(),
				)
			}
		} else {
			if err := s.dispatcher.Dispatch(ctx, NewAddMemberCmd(survivorActor, mem.TenantID, merged)); err != nil {
				return classifyDispatchError(err, survivorID,
					"tenant_id", mem.TenantID.Get(),
					"loser_id", loserID.Get().String(),
				)
			}
			survivorRoles[mem.TenantID] = merged
		}
		loserActor := ActorIDFromUser(loserID)
		if err := s.dispatcher.Dispatch(ctx, NewRemoveMemberCmd(loserActor, mem.TenantID)); err != nil {
			return classifyDispatchError(err, loserID,
				"tenant_id", mem.TenantID.Get(),
				"survivor_id", survivorID.Get().String(),
			)
		}
	}
	return nil
}

// unionRoles returns the ordered union of two role sets: every role of a
// followed by every role of b that a does not hold.
func unionRoles(a, b []Role) []Role {
	merged := slices.Clone(a)
	for _, role := range b {
		if !slices.Contains(merged, role) {
			merged = append(merged, role)
		}
	}
	return merged
}

// DuplicateUserGroup names one cluster of duplicate users. A cluster is
// either every live user sharing one email (Key = the email, Account empty)
// or every live user sharing one external account (Account = "provider/subject",
// Key empty). The admin picks a survivor and calls [Service.MergeUsers].
type DuplicateUserGroup struct {
	Key     string  `json:"key,omitempty"`
	Account string  `json:"account,omitempty"`
	Users   []*User `json:"users"`
}

// DuplicateUsers reports every cluster of live users a human would call the
// same person: identical emails, or one identity-provider subject linked to
// more than one row. Emails compare after lowercasing and trimming; the live
// deployment's nine-row human (issue #46) surfaced through exactly these
// signals.
func (s *Service) DuplicateUsers(_ context.Context) []DuplicateUserGroup {
	byEmail := make(map[string][]*User)
	byAccount := make(map[string][]*User)
	for _, user := range s.readModel.AllUsers() {
		emailKey := normalizeEmailKey(user.Email)
		byEmail[emailKey] = append(byEmail[emailKey], user)
		for _, account := range user.ExternalAccounts {
			key := account.Provider + "/" + account.Subject
			byAccount[key] = append(byAccount[key], user)
		}
	}
	groups := make([]DuplicateUserGroup, 0)
	for email, users := range byEmail {
		if len(users) > 1 {
			groups = append(groups, DuplicateUserGroup{Key: email, Account: "", Users: users})
		}
	}
	for account, users := range byAccount {
		if len(users) > 1 {
			groups = append(groups, DuplicateUserGroup{Key: "", Account: account, Users: users})
		}
	}
	slices.SortFunc(groups, func(a, b DuplicateUserGroup) int {
		if a.Account != b.Account {
			return strings.Compare(a.Account, b.Account)
		}
		return strings.Compare(a.Key, b.Key)
	})
	return groups
}

func normalizeEmailKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
