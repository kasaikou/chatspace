package chatspace

import "github.com/gammazero/deque"

type ScheduleEvent struct {
	Unix int64
	Func func()
}

type ScheduleQueue deque.Deque[ScheduleEvent]

func NewScheduleQueue() *ScheduleQueue {
	return (*ScheduleQueue)(deque.New[ScheduleEvent]())
}

func (sq *ScheduleQueue) super() *deque.Deque[ScheduleEvent] {
	return (*deque.Deque[ScheduleEvent])(sq)
}

func (sq *ScheduleQueue) Len() int {
	return sq.super().Len()
}

func (sq *ScheduleQueue) Less(i, j int) bool {
	return sq.super().At(i).Unix < sq.super().At(j).Unix
}

func (sq *ScheduleQueue) Swap(i, j int) {
	queue := sq.super()
	iEvent, jEvent := queue.At(i), queue.At(j)
	queue.Set(i, jEvent)
	queue.Set(j, iEvent)
}

func (sq *ScheduleQueue) PushBack(event ScheduleEvent) {
	sq.super().PushBack(event)
}

func (sq *ScheduleQueue) Front() ScheduleEvent {
	return sq.super().Front()
}

func (sq *ScheduleQueue) PopFront() {
	sq.super().PopFront()
}
