package event

import (
	"io"
	"reflect"
)

// SubscriptionOpt represents a subscriber option. Use the options exposed by the implementation of choice.
type SubscriptionOpt = func(interface{}) error

// EmitterOpt represents an emitter option. Use the options exposed by the implementation of choice.
type EmitterOpt = func(interface{}) error

// CancelFunc closes a subscriber.
type CancelFunc = func()

// wildcardSubscriptionType is a virtual type to represent wildcard
// subscriptions.
type wildcardSubscriptionType interface{}

// WildcardSubscription is the type to subscribe to to receive all events
// emitted in the eventbus.
var WildcardSubscription = new(wildcardSubscriptionType)

// Emitter represents an actor that emits events onto the eventbus.
type Emitter interface {
	io.Closer

	// Emit emits an event onto the eventbus. If any channel subscribed to the topic is blocked,
	// calls to Emit will block.
	//
	// Calling this function with wrong event type will cause a panic.
	Emit(evt interface{}) error
}

// Subscription represents a subscription to one or multiple event types. 订阅
type Subscription interface {
	io.Closer

	// Out returns the channel from which to consume events. 消耗事件通道
	Out() <-chan interface{}
}

// Bus is an interface for a type-based event delivery system. 基于时间的传输系统bus接口
type Bus interface {
	// Subscribe creates a new Subscription.
	//
	// eventType can be either a pointer to a single event type, or a slice of pointers to
	// subscribe to multiple event types at once, under a single subscription (and channel).
	//
	// Failing to drain the channel may cause publishers to block.
	//
	// If you want to subscribe to ALL events emitted in the bus, use
	// `WildcardSubscription` as the `eventType`:
	//
	//  eventbus.Subscribe(WildcardSubscription)
	//
	// Simple example
	//
	//  sub, err := eventbus.Subscribe(new(EventType))
	//  defer sub.Close()
	//  for e := range sub.Out() {
	//    event := e.(EventType) // guaranteed safe
	//    [...]
	//  }
	//
	// Multi-type example
	//
	//  sub, err := eventbus.Subscribe([]interface{}{new(EventA), new(EventB)})
	//  defer sub.Close()
	//  for e := range sub.Out() {
	//    select e.(type):
	//      case EventA:
	//        [...]
	//      case EventB:
	//        [...]
	//    }
	//  }
	// 订阅事件类型
	Subscribe(eventType interface{}, opts ...SubscriptionOpt) (Subscription, error)

	// Emitter creates a new event emitter.
	// 创建一个新事件的Emitter
	// eventType accepts typed nil pointers, and uses the type information for wiring purposes.
	//
	// Example:
	//  em, err := eventbus.Emitter(new(EventT))
	//  defer em.Close() // MUST call this after being done with the emitter
	//  em.Emit(EventT{})
	Emitter(eventType interface{}, opts ...EmitterOpt) (Emitter, error)

	// GetAllEventTypes returns all the event types that this bus knows about
	// (having emitters and subscribers). It omits the WildcardSubscription.
	// 返回所有当前bus知道的事件类型
	// The caller is guaranteed that this function will only return value types;
	// no pointer types will be returned.
	GetAllEventTypes() []reflect.Type
}
