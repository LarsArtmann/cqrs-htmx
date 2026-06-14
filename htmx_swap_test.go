package cqrshtmx_test

import (
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HTMX Swap Strategy", func() {
	It("has correct swap strategy values", func() {
		Expect(string(cqrshtmx.SwapInnerHTML)).To(Equal("innerHTML"))
		Expect(string(cqrshtmx.SwapOuterHTML)).To(Equal("outerHTML"))
		Expect(string(cqrshtmx.SwapBeforeBegin)).To(Equal("beforebegin"))
		Expect(string(cqrshtmx.SwapAfterBegin)).To(Equal("afterbegin"))
		Expect(string(cqrshtmx.SwapBeforeEnd)).To(Equal("beforeend"))
		Expect(string(cqrshtmx.SwapAfterEnd)).To(Equal("afterend"))
		Expect(string(cqrshtmx.SwapDelete)).To(Equal("delete"))
		Expect(string(cqrshtmx.SwapNone)).To(Equal("none"))
	})

	It("validates known strategies", func() {
		Expect(cqrshtmx.SwapInnerHTML.Valid()).To(BeTrue())
		Expect(cqrshtmx.SwapOuterHTML.Valid()).To(BeTrue())
		Expect(cqrshtmx.SwapBeforeBegin.Valid()).To(BeTrue())
		Expect(cqrshtmx.SwapAfterBegin.Valid()).To(BeTrue())
		Expect(cqrshtmx.SwapBeforeEnd.Valid()).To(BeTrue())
		Expect(cqrshtmx.SwapAfterEnd.Valid()).To(BeTrue())
		Expect(cqrshtmx.SwapDelete.Valid()).To(BeTrue())
		Expect(cqrshtmx.SwapNone.Valid()).To(BeTrue())
	})

	It("rejects unknown strategies", func() {
		Expect(cqrshtmx.SwapStrategy("invalid").Valid()).To(BeFalse())
		Expect(cqrshtmx.SwapStrategy("").Valid()).To(BeFalse())
	})
})
