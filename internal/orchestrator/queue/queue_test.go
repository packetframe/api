package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueueAddLenPop(t *testing.T) {
	q := Queue{}
	q.Add("foo")
	q.Add("foo") // Duplicate
	q.Add("bar")
	assert.Equal(t, 2, q.Len())
	e := q.Pop()
	assert.Equal(t, "foo", e)
	assert.Equal(t, 1, q.Len())
}
