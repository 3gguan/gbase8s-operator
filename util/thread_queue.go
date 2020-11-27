package util

type ThreadQueue struct {
	msg chan interface{}
}

func NewThreadQueue(maxSize int) *ThreadQueue {
	queue := ThreadQueue{
		msg: make(chan interface{}, maxSize),
	}

	return &queue
}

func (q *ThreadQueue) Add(element interface{}) {
	q.msg <- element
}

func (q *ThreadQueue) Get() interface{} {
	return <-q.msg
}
