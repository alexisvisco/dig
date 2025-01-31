// Copyright (c) 2021 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dig_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/alexisvisco/dig"
	"github.com/alexisvisco/dig/internal/digtest"
	"github.com/alexisvisco/dig/internal/dot"
	"github.com/stretchr/testify/assert"
)

func TestDotGraph(t *testing.T) {
	tparam := func(t reflect.Type, n string, g string, o bool) *dot.Param {
		return &dot.Param{
			Node: &dot.Node{
				Type:  t,
				Name:  n,
				Group: g,
			},
			Optional: o,
		}
	}

	tresult := func(t reflect.Type, n string, g string, gi int) *dot.Result {
		return &dot.Result{
			Node: &dot.Node{
				Type:  t,
				Name:  n,
				Group: g,
			},
			GroupIndex: gi,
		}
	}

	type t1 struct{}
	type t2 struct{}
	type t3 struct{}
	type t4 struct{}
	type t5 strings.Reader

	type1 := reflect.TypeOf(t1{})
	type2 := reflect.TypeOf(t2{})
	type3 := reflect.TypeOf(t3{})
	type4 := reflect.TypeOf(t4{})
	type5 := reflect.TypeOf(t5{})
	type6 := reflect.Indirect(reflect.ValueOf(new(io.Reader))).Type()
	type7 := reflect.Indirect(reflect.ValueOf(new(io.Writer))).Type()

	p1 := tparam(type1, "", "", false)
	p2 := tparam(type2, "", "", false)
	p3 := tparam(type3, "", "", false)
	p4 := tparam(type4, "", "", false)

	r1 := tresult(type1, "", "", 0)
	r2 := tresult(type2, "", "", 0)
	r3 := tresult(type3, "", "", 0)
	r4 := tresult(type4, "", "", 0)
	r5 := tresult(type5, "", "", 0)
	r6 := tresult(type6, "", "", 0)

	t.Parallel()

	t.Run("create graph with one constructor", func(t *testing.T) {
		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{p1},
				Results: []*dot.Result{r2},
			},
		}

		c := digtest.New(t)
		c.Provide(func(A t1) t2 { return t2{} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("create graph with one constructor and as interface option", func(t *testing.T) {
		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{p1},
				Results: []*dot.Result{r5, r6},
			},
		}

		c := digtest.New(t)
		c.Provide(func(A t1) t5 { return t5{} }, dig.As(new(io.Reader)))

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("create graph with multple constructors", func(t *testing.T) {
		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{p1},
				Results: []*dot.Result{r2},
			},
			{
				Params:  []*dot.Param{p1},
				Results: []*dot.Result{r3},
			},
			{
				Params:  []*dot.Param{p2},
				Results: []*dot.Result{r4},
			},
		}

		c := digtest.New(t)
		c.Provide(func(A t1) t2 { return t2{} })
		c.Provide(func(A t1) t3 { return t3{} })
		c.Provide(func(A t2) t4 { return t4{} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("constructor with multiple params and results", func(t *testing.T) {
		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{p3, p4},
				Results: []*dot.Result{r1, r2},
			},
		}

		c := digtest.New(t)
		c.Provide(func(A t3, B t4) (t1, t2) { return t1{}, t2{} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("param objects and result objects", func(t *testing.T) {
		type in struct {
			dig.In

			A t1
			B t2
		}

		type out struct {
			dig.Out

			C t3
			D t4
		}

		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{p1, p2},
				Results: []*dot.Result{r3, r4},
			},
		}

		c := digtest.New(t)
		c.Provide(func(i in) out { return out{} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("nested param object", func(t *testing.T) {
		type in struct {
			dig.In

			A    t1
			Nest struct {
				dig.In

				B    t2
				Nest struct {
					dig.In

					C t3
				}
			}
		}

		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{p1, p2, p3},
				Results: []*dot.Result{r4},
			},
		}

		c := digtest.New(t)
		c.Provide(func(p in) t4 { return t4{} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("nested result object", func(t *testing.T) {
		type nested1 struct {
			dig.Out

			D t4
		}

		type nested2 struct {
			dig.Out

			C    t3
			Nest nested1
		}

		type out struct {
			dig.Out

			B    t2
			Nest nested2
		}

		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{p1},
				Results: []*dot.Result{r2, r3, r4},
			},
		}

		c := digtest.New(t)
		c.Provide(func(A t1) out { return out{} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("value groups", func(t *testing.T) {
		type in struct {
			dig.In

			D []t1 `group:"foo"`
		}

		type out1 struct {
			dig.Out

			A t1 `group:"foo"`
		}

		type out2 struct {
			dig.Out

			A t1 `group:"foo"`
		}

		res0 := tresult(type1, "", "foo", 0)
		res1 := tresult(type1, "", "foo", 1)

		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{p2},
				Results: []*dot.Result{res0},
			},
			{
				Params:  []*dot.Param{p4},
				Results: []*dot.Result{res1},
			},
			{
				GroupParams: []*dot.Group{
					{
						Type:    type1,
						Name:    "foo",
						Results: []*dot.Result{res0, res1},
					},
				},
				Results: []*dot.Result{r3},
			},
		}

		c := digtest.New(t)
		c.Provide(func(B t2) out1 { return out1{} })
		c.Provide(func(B t4) out2 { return out2{} })
		c.Provide(func(i in) t3 { return t3{} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("value groups as", func(t *testing.T) {
		c := digtest.New(t)
		c.Provide(
			func() *bytes.Buffer { return bytes.NewBufferString("foo") },
			dig.As(new(io.Reader), new(io.Writer)),
			dig.Group("buffs"),
		)
		c.Provide(
			func() *bytes.Buffer { return bytes.NewBufferString("bar") },
			dig.As(new(io.Reader), new(io.Writer)),
			dig.Group("buffs"),
		)
		expected := []*dot.Ctor{
			{
				Results: []*dot.Result{tresult(type6, "", "buffs", 0), tresult(type7, "", "buffs", 0)},
			},
			{
				Results: []*dot.Result{tresult(type6, "", "buffs", 1), tresult(type7, "", "buffs", 1)},
			},
		}
		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("named values", func(t *testing.T) {
		type in struct {
			dig.In

			A t1 `name:"A"`
		}

		type out struct {
			dig.Out

			B t2 `name:"B"`
		}

		expected := []*dot.Ctor{
			{
				Params: []*dot.Param{
					tparam(type1, "A", "", false),
				},
				Results: []*dot.Result{
					tresult(type2, "B", "", 0),
				},
			},
		}

		c := digtest.New(t)
		c.Provide(func(i in) out { return out{B: t2{}} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})

	t.Run("optional dependencies", func(t *testing.T) {
		type in struct {
			dig.In

			A t1 `name:"A" optional:"true"`
			B t2 `name:"B"`
			C t3 `optional:"true"`
		}

		par1 := tparam(type1, "A", "", true)
		par2 := tparam(type2, "B", "", false)
		par3 := tparam(type3, "", "", true)

		expected := []*dot.Ctor{
			{
				Params:  []*dot.Param{par1, par2, par3},
				Results: []*dot.Result{r4},
			},
		}

		c := digtest.New(t)
		c.Provide(func(i in) t4 { return t4{} })

		dg := c.CreateGraph()
		assertCtorsEqual(t, expected, dg.Ctors)
	})
}

func assertCtorEqual(t *testing.T, expected *dot.Ctor, ctor *dot.Ctor) {
	assert.Equal(t, expected.Params, ctor.Params)
	assert.Equal(t, expected.Results, ctor.Results)
	assert.NotZero(t, ctor.Line)
}

func assertCtorsEqual(t *testing.T, expected []*dot.Ctor, ctors []*dot.Ctor) {
	for i, c := range ctors {
		assertCtorEqual(t, expected[i], c)
	}
}

func TestVisualize(t *testing.T) {
	type t1 struct{}
	type t2 struct{}
	type t3 struct{}
	type t4 struct{}

	t.Parallel()

	t.Run("empty graph in container", func(t *testing.T) {
		c := digtest.New(t)
		dig.VerifyVisualization(t, "empty", c.Container)
	})

	t.Run("simple graph", func(t *testing.T) {
		c := digtest.New(t)

		c.Provide(func() (t1, t2) { return t1{}, t2{} })
		c.Provide(func(A t1, B t2) (t3, t4) { return t3{}, t4{} })
		dig.VerifyVisualization(t, "simple", c.Container)
	})

	t.Run("named types", func(t *testing.T) {
		c := digtest.New(t)

		type in struct {
			dig.In

			A t3 `name:"foo"`
		}
		type out1 struct {
			dig.Out

			A t1 `name:"bar"`
			B t2 `name:"baz"`
		}
		type out2 struct {
			dig.Out

			A t3 `name:"foo"`
		}

		c.Provide(func(in) out1 { return out1{} })
		c.Provide(func() out2 { return out2{} })
		dig.VerifyVisualization(t, "named", c.Container)
	})

	t.Run("dig.As two types", func(t *testing.T) {
		c := digtest.New(t)

		c.RequireProvide(
			func() *bytes.Buffer {
				panic("this function should not be called")
			},
			dig.As(new(io.Reader), new(io.Writer)))

		dig.VerifyVisualization(t, "dig_as_two", c.Container)
	})

	t.Run("optional params", func(t *testing.T) {
		c := digtest.New(t)

		type in struct {
			dig.In

			A t1 `optional:"true"`
		}

		c.Provide(func() t1 { return t1{} })
		c.Provide(func(in) t2 { return t2{} })
		dig.VerifyVisualization(t, "optional", c.Container)
	})

	t.Run("grouped types", func(t *testing.T) {
		c := digtest.New(t)

		type in struct {
			dig.In

			A []t3 `group:"foo"`
		}

		type out1 struct {
			dig.Out

			A t3 `group:"foo"`
		}

		type out2 struct {
			dig.Out

			A t3 `group:"foo"`
		}

		c.Provide(func() out1 { return out1{} })
		c.Provide(func() out2 { return out2{} })
		c.Provide(func(in) t2 { return t2{} })

		dig.VerifyVisualization(t, "grouped", c.Container)
	})

	t.Run("constructor fails with an error", func(t *testing.T) {
		c := digtest.New(t)

		type in1 struct {
			dig.In

			C []t1 `group:"g1"`
		}

		type in2 struct {
			dig.In

			A []t2 `group:"g2"`
			B t3   `name:"n3"`
		}

		type out1 struct {
			dig.Out

			B t3 `name:"n3"`
			C t2 `group:"g2"`
		}

		type out2 struct {
			dig.Out

			D t2 `group:"g2"`
		}

		type out3 struct {
			dig.Out

			A t1 `group:"g1"`
			B t2 `group:"g2"`
		}

		c.Provide(func(in1) out1 { return out1{} })
		c.Provide(func(in2) t4 { return t4{} })
		c.Provide(func() out2 { return out2{} })
		c.Provide(func() (out3, error) { return out3{}, errors.New("great sadness") })
		err := c.Invoke(func(t4 t4) {})

		dig.VerifyVisualization(t, "error", c.Container, dig.VisualizeError(err))

		t.Run("non-failing graph nodes are pruned", func(t *testing.T) {
			t.Run("prune non-failing constructor result", func(t *testing.T) {
				c := digtest.New(t)
				c.Provide(func(in1) out1 { return out1{} })
				c.Provide(func(in2) t4 { return t4{} })
				c.Provide(func() (out2, error) { return out2{}, errors.New("great sadness") })
				c.Provide(func() out3 { return out3{} })
				err := c.Invoke(func(t4 t4) {})

				dig.VerifyVisualization(t, "prune_constructor_result", c.Container, dig.VisualizeError(err))
			})

			t.Run("if only the root node fails all node except for the root should be pruned", func(t *testing.T) {
				c := digtest.New(t)
				c.Provide(func(in1) out1 { return out1{} })
				c.Provide(func(in2) (t4, error) { return t4{}, errors.New("great sadness") })
				c.Provide(func() out2 { return out2{} })
				c.Provide(func() out3 { return out3{} })
				err := c.Invoke(func(t4 t4) {})

				dig.VerifyVisualization(t, "prune_non_root_nodes", c.Container, dig.VisualizeError(err))
			})
		})
	})

	t.Run("missing types", func(t *testing.T) {
		c := digtest.New(t)

		c.Provide(func(A t1, B t2, C t3) t4 { return t4{} })
		err := c.Invoke(func(t4 t4) {})

		dig.VerifyVisualization(t, "missing", c.Container, dig.VisualizeError(err))
	})

	t.Run("missing dependency", func(t *testing.T) {
		c := digtest.New(t)
		err := c.Invoke(func(t1 t1) {})

		dig.VerifyVisualization(t, "missingDep", c.Container, dig.VisualizeError(err))
	})
}

func TestVisualizeErrorString(t *testing.T) {
	t.Parallel()

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		opt := dig.VisualizeError(nil)
		assert.Equal(t, "VisualizeError(<nil>)", fmt.Sprint(opt))
	})

	t.Run("not nil", func(t *testing.T) {
		t.Parallel()

		opt := dig.VisualizeError(errors.New("great sadness"))
		assert.Equal(t, "VisualizeError(great sadness)", fmt.Sprint(opt))
	})
}
