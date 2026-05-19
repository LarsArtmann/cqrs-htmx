package usermgmt

import (
	"fmt"
	"time"
)

const defaultSessionTTL = 24 * time.Hour

type Service struct {
	authz      *Authz
	users      UserStore
	sessions   SessionStore
	sessionTTL time.Duration
}

type ServiceConfig struct {
	Authz        *Authz
	UserStore    UserStore
	SessionStore SessionStore
	SessionTTL   time.Duration
}

func NewService(cfg ServiceConfig) (*Service, error) {
	if cfg.UserStore == nil {
		cfg.UserStore = NewInMemoryUserStore()
	}
	if cfg.SessionStore == nil {
		cfg.SessionStore = NewInMemorySessionStore()
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = defaultSessionTTL
	}
	if cfg.Authz == nil {
		a, err := NewAuthz()
		if err != nil {
			return nil, fmt.Errorf("create authz: %w", err)
		}
		cfg.Authz = a
	}

	return &Service{
		authz:      cfg.Authz,
		users:      cfg.UserStore,
		sessions:   cfg.SessionStore,
		sessionTTL: cfg.SessionTTL,
	}, nil
}

func (s *Service) Authz() *Authz { return s.authz }

type RegisterRequest struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type RegisterResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

func (s *Service) Register(req RegisterRequest) (*RegisterResponse, error) {
	existing, err := s.users.FindByEmail(req.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailExists
	}

	user := NewUser(req.ID, req.Email, req.DisplayName)
	if err := user.SetPassword(req.Password); err != nil {
		return nil, fmt.Errorf("set password: %w", err)
	}

	user.AddRole(RoleUser)

	if err := s.users.Save(user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}

	if err := s.authz.AddGroupPolicy(GroupPolicy{
		User: user.ID, Role: RoleUser, Domain: user.ID,
	}); err != nil {
		return nil, fmt.Errorf("assign role: %w", err)
	}

	session, err := s.sessions.Create(user.ID, s.sessionTTL)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &RegisterResponse{User: user, Session: session}, nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User    *User    `json:"user"`
	Session *Session `json:"session"`
}

func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	user, err := s.users.FindByEmail(req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.CheckPassword(req.Password) {
		return nil, ErrInvalidCredentials
	}

	session, err := s.sessions.Create(user.ID, s.sessionTTL)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &LoginResponse{User: user, Session: session}, nil
}

func (s *Service) Logout(token string) error {
	return s.sessions.Delete(token)
}

func (s *Service) Authenticate(token string) (*User, error) {
	session, err := s.sessions.Find(token)
	if err != nil {
		return nil, ErrUnauthorized
	}

	if session.IsExpired() {
		_ = s.sessions.Delete(token)
		return nil, ErrSessionExpired
	}

	if !session.Valid(token) {
		return nil, ErrUnauthorized
	}

	user, err := s.users.FindByID(session.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *Service) Authorize(sub, dom, obj string, act Action) error {
	return s.authz.Authorize(sub, dom, obj, act)
}

func (s *Service) GetUser(id string) (*User, error) {
	return s.users.FindByID(id)
}

func (s *Service) UpdateRoles(userID string, roles []string, domain string) error {
	user, err := s.users.FindByID(userID)
	if err != nil {
		return err
	}

	currentRoles, err := s.authz.RolesForUser(userID, domain)
	if err != nil {
		return fmt.Errorf("get roles: %w", err)
	}

	for _, role := range currentRoles {
		if err := s.authz.RemoveGroupPolicy(GroupPolicy{
			User: userID, Role: role, Domain: domain,
		}); err != nil {
			return fmt.Errorf("revoke role %s: %w", role, err)
		}
	}

	for _, role := range roles {
		if err := s.authz.AddGroupPolicy(GroupPolicy{
			User: userID, Role: role, Domain: domain,
		}); err != nil {
			return fmt.Errorf("assign role %s: %w", role, err)
		}
	}

	user.Roles = roles
	user.UpdatedAt = time.Now().UTC()
	return s.users.Save(user)
}
