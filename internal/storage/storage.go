package storage

import "errors"

var (
	ErrServiceNotFound           = errors.New("service not found")
	ErrServiceNameExists         = errors.New("service name already exists")
	ErrServiceInUse              = errors.New("service is referenced by subscriptions")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists")
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionNotCreated    = errors.New("subscription not created")
)
