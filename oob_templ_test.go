package cqrshtmx_test

import (
	"context"
	"io"
	"strings"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// renderComponent runs a component into a string, failing the spec on error.
func renderComponent(c cqrshtmx.TemplComponent) string {
	var out strings.Builder
	Expect(c.Render(context.Background(), &out)).To(Succeed())

	return out.String()
}

var _ = Describe("OOBTempl", func() {
	It("emits the same bytes as OOBHTML with the default swap strategy", func() {
		component := cqrshtmx.OOBTempl("counter", &bddTemplComponent{html: "<span>3</span>"})

		Expect(renderComponent(component)).To(Equal(cqrshtmx.OOBHTML("counter", "<span>3</span>")))
	})

	It("emits the same bytes as OOBHTML with a custom swap strategy", func() {
		component := cqrshtmx.OOBTempl("list", &bddTemplComponent{html: "<li>item</li>"},
			cqrshtmx.SwapBeforeEnd)

		Expect(renderComponent(component)).To(Equal(cqrshtmx.OOBHTML("list", "<li>item</li>", cqrshtmx.SwapBeforeEnd)))
	})

	It("passes through a component that already carries hx-swap-oob", func() {
		inner := `<div id="x" hx-swap-oob="true">already tagged</div>`

		Expect(renderComponent(cqrshtmx.OOBTempl("x", &bddTemplComponent{html: inner}))).To(Equal(inner))
	})

	It("escapes the id so it cannot break out of the attribute", func() {
		got := renderComponent(cqrshtmx.OOBTempl(`x" onmouseover="alert(1)`,
			&bddTemplComponent{html: "<p>hi</p>"}))

		Expect(got).NotTo(ContainSubstring(`x" onmouseover`))
		Expect(got).To(ContainSubstring("&#34;"))
	})

	It("wraps an empty id", func() {
		Expect(
			renderComponent(cqrshtmx.OOBTempl("", &bddTemplComponent{html: "<span>x</span>"})),
		).To(Equal(`<div id="" hx-swap-oob="true"><span>x</span></div>`))
	})

	It("falls back to the default swap when given an empty strategy", func() {
		Expect(
			renderComponent(cqrshtmx.OOBTempl("slot", &bddTemplComponent{html: "x"}, "")),
		).To(Equal(`<div id="slot" hx-swap-oob="true">x</div>`))
	})

	It("propagates the inner render error and writes nothing", func() {
		var out strings.Builder

		err := cqrshtmx.OOBTempl("slot", errTemplComponent{}).
			Render(context.Background(), &out)

		Expect(err).To(MatchError("disk full"))
		Expect(out.String()).To(BeEmpty())
	})

	It("renders a component with a failing writer", func() {
		err := cqrshtmx.OOBTempl("slot", &bddTemplComponent{html: "x"}).
			Render(context.Background(), failingWriter{})

		Expect(err).To(MatchError(io.ErrShortWrite))
	})
})

// failingWriter refuses every write, exercising the envelope's error paths.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, io.ErrShortWrite
}
