package htmx_test

import (
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMX Swap Strategy", func() {
	It("has correct swap strategy values", func() {
		Expect(string(htmx.SwapInnerHTML)).To(Equal("innerHTML"))
		Expect(string(htmx.SwapOuterHTML)).To(Equal("outerHTML"))
		Expect(string(htmx.SwapBeforeBegin)).To(Equal("beforebegin"))
		Expect(string(htmx.SwapAfterBegin)).To(Equal("afterbegin"))
		Expect(string(htmx.SwapBeforeEnd)).To(Equal("beforeend"))
		Expect(string(htmx.SwapAfterEnd)).To(Equal("afterend"))
		Expect(string(htmx.SwapDelete)).To(Equal("delete"))
		Expect(string(htmx.SwapNone)).To(Equal("none"))
	})

	It("validates known strategies", func() {
		Expect(htmx.SwapInnerHTML.Valid()).To(BeTrue())
		Expect(htmx.SwapOuterHTML.Valid()).To(BeTrue())
		Expect(htmx.SwapBeforeBegin.Valid()).To(BeTrue())
		Expect(htmx.SwapAfterBegin.Valid()).To(BeTrue())
		Expect(htmx.SwapBeforeEnd.Valid()).To(BeTrue())
		Expect(htmx.SwapAfterEnd.Valid()).To(BeTrue())
		Expect(htmx.SwapDelete.Valid()).To(BeTrue())
		Expect(htmx.SwapNone.Valid()).To(BeTrue())
	})

	It("rejects unknown strategies", func() {
		Expect(htmx.SwapStrategy("invalid").Valid()).To(BeFalse())
		Expect(htmx.SwapStrategy("").Valid()).To(BeFalse())
	})
})
