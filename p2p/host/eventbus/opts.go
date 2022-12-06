package eventbus

type subSettings struct {
	buffer int
}

var subSettingsDefault = subSettings{
	buffer: 16,
}

func BufSize(n int) func(interface{}) error {
	return func(s interface{}) error {
		s.(*subSettings).buffer = n
		return nil
	}
}

type emitterSettings struct {
	makeStateful bool
}

// Stateful is an Emitter option which makes the eventbus channel
// 'remember' last event sent, and when a new subscriber joins the
// bus, the remembered event is immediately sent to the subscription
// channel.
// Stateful是事件发射器选项，用于记住eventbus通道上次发送的事件
// 当新的订阅者加入时，记住的事件将会发送给通道
// This allows to provide state tracking for dynamic systems, and/or
// allows new subscribers to verify that there are Emitters on the channel
// 提供动态状态追踪系统，允许新的订阅者验证通道发出的事件
func Stateful(s interface{}) error {
	s.(*emitterSettings).makeStateful = true
	return nil
}
