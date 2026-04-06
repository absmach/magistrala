// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"context"
	"testing"

	"github.com/absmach/magistrala/notifications"
	"github.com/absmach/magistrala/notifications/middleware"
	"github.com/absmach/magistrala/notifications/mocks"
	"github.com/go-kit/kit/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCounter struct {
	mock.Mock
	metrics.Counter
}

func (m *mockCounter) Add(delta float64) {
	m.Called(delta)
}

func (m *mockCounter) With(labelValues ...string) metrics.Counter {
	args := m.Called(labelValues)
	return args.Get(0).(metrics.Counter)
}

type mockHistogram struct {
	mock.Mock
	metrics.Histogram
}

func (m *mockHistogram) Observe(value float64) {
	m.Called(value)
}

func (m *mockHistogram) With(labelValues ...string) metrics.Histogram {
	args := m.Called(labelValues)
	return args.Get(0).(metrics.Histogram)
}

func TestMetricsMiddleware(t *testing.T) {
	notifier := new(mocks.Notifier)
	counter := new(mockCounter)
	histogram := new(mockHistogram)

	counter.On("With", mock.Anything).Return(counter)
	counter.On("Add", mock.Anything).Return()
	histogram.On("With", mock.Anything).Return(histogram)
	histogram.On("Observe", mock.Anything).Return()

	mm := middleware.NewMetrics(notifier, counter, histogram)

	notif := notifications.Notification{
		Type:       notifications.Invitation,
		InviterID:  "inv1",
		InviteeID:  "inv2",
		DomainID:   "dom1",
		DomainName: "Domain",
		RoleID:     "role1",
		RoleName:   "Admin",
	}

	notifier.On("Notify", mock.Anything, notif).Return(nil).Once()

	err := mm.Notify(context.Background(), notif)
	assert.NoError(t, err)
	notifier.AssertExpectations(t)
	counter.AssertCalled(t, "With", []string{"method", "send_invitation_notification"})
	counter.AssertCalled(t, "Add", mock.Anything)
	histogram.AssertCalled(t, "With", []string{"method", "send_invitation_notification"})
	histogram.AssertCalled(t, "Observe", mock.Anything)
}
