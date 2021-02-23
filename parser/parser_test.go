package parser

import (
	"cdr.dev/slog/sloggers/slogtest"
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

type (
	parserTestCase struct {
		name string
		input string
		expected string
	}
)

var cases = []parserTestCase {
	{
		name: "lists 1",
		input: `* Item 1
** Item 1.1
+
image::image1.png[]
+
More text.
+
NOTE: Admonition text.
+
** Item 1.2`,
expected: `
container block:
  list begin: 0, false
  item:
    container block:
      container block:
        text: Item 1
      list begin: 1, false
      item:
        container block:
          container block:
            text: Item 1.1
          image: image1.png
          container block:
            text: More text.
          admonition: NOTE
            container block:
              container block:
                text: Admonition text.
      item:
        container block:
          container block:
            text: Item 1.2
      list end
  list end`,
	},
	{
		name: "lists 2",
		input: `. Item 1
* Item 1.1
. Item 2`,
		expected: ``,
	},
}


func TestParser(t *testing.T) {
	logger := slogtest.Make(t, nil)
	logger.Info(context.Background(), "log message")

	for _, tc := range cases {
		p := New(tc.input, logger)
		doc, err := p.Parse()
		t.Log(doc.String(""))
		assert.Nil(t, err)
		assert.Equal(t, tc.expected, doc.String(""))

	}
}

/*func TestDebug(t *testing.T) {
	logger := slogtest.Make(t, nil)
	logger.Info(context.Background(), "log message")

	for _, tc := range cases[len(cases)-1:] {
		p := New(tc.input, logger)
		doc, err := p.Parse()

		t.Log("\n" + doc.String(""))
		if err != nil {
			assert.Failf(t, "got errors during parsing", "%v", err)
		}


	}
}*/

