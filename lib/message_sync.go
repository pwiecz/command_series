package lib

type MessageSync struct {
	update chan interface{}
	cont   chan bool
}

func NewMessageSync() *MessageSync {
	return &MessageSync{
		update: make(chan interface{}),
		cont:   make(chan bool)}
}

func (s *MessageSync) SendUpdate(msg interface{}) bool {
	s.update <- msg
	return <-s.cont
}
func (s *MessageSync) Wait() bool {
	return <-s.cont
}
func (s *MessageSync) GetUpdate() interface{} {
	s.cont <- true
	return <-s.update
}
func (s *MessageSync) Stop() {
	s.cont <- false
}
