package queue

import "github.com/packetframe/api/internal/common/util"

type Queue struct {
	elems []string
}

// Add adds an item to the end of the queue
func (q *Queue) Add(elem string) {
	if !util.StrSliceContains(q.elems, elem) {
		q.elems = append(q.elems, elem)
	}
}

// Len gets the length of the queue
func (q *Queue) Len() int {
	return len(q.elems)
}

// Pop returns the first element from the queue and removes it
func (q *Queue) Pop() string {
	if q.Len() == 0 {
		return ""
	}

	elem := q.elems[0]
	q.elems = q.elems[1:]
	return elem
}
