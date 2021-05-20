package client

// nilNotifier implements a Notify method that does nothing
type nilNotifier struct {
}

func (n *nilNotifier) Notify() {
}
