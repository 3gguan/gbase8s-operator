package util

type Thread struct {
	Queue    *ThreadQueue
	callback func(interface{})
}

func NewThread(queue *ThreadQueue, f func(interface{})) *Thread {
	return &Thread{
		Queue:    queue,
		callback: f,
	}
}

func (t *Thread) Run() *Thread {
	go func() {
		for {
			qmsg := t.Queue.Get()
			t.callback(qmsg)
		}
	}()

	return t
}
