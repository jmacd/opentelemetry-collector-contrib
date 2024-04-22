// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	connectURL = "amqp://localhost"
	exchange   = "amq.direct"
	routingKey = "some_routing_key"
)

func TestConnectAndClose(t *testing.T) {
	client := mockClient{}
	connection := mockConnection{}
	dialConfig := DialConfig{
		URL: connectURL,
	}

	// Start the connection successfully
	client.On("DialConfig", connectURL, mock.Anything).Return(&connection, nil)
	connection.On("NotifyClose", mock.Anything).Return(make(chan *amqp.Error))

	publisher, err := NewConnection(zap.NewNop(), &client, dialConfig)

	require.NoError(t, err)
	client.AssertExpectations(t)

	// Close the connection
	connection.On("IsClosed").Return(false)
	connection.On("Close").Return(nil)

	err = publisher.Close()
	require.NoError(t, err)
	client.AssertExpectations(t)
	connection.AssertExpectations(t)
}

func TestConnectionErrorAndClose(t *testing.T) {
	client := mockClient{}
	dialConfig := DialConfig{
		URL: connectURL,
	}

	client.On("DialConfig", connectURL, mock.Anything).Return(nil, errors.New("simulated connection error"))
	publisher, err := NewConnection(zap.NewNop(), &client, dialConfig)

	assert.EqualError(t, err, "simulated connection error")

	err = publisher.Close()
	require.NoError(t, err)

	client.AssertExpectations(t)
}

func TestPublishAckedWithinTimeout(t *testing.T) {
	client, connection, channel, confirmation := setupMocksForSuccessfulPublish()

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	err = publisher.Publish(context.Background(), makePublishMessage())

	require.NoError(t, err)
	client.AssertExpectations(t)
	connection.AssertExpectations(t)
	channel.AssertExpectations(t)
	confirmation.AssertExpectations(t)
}

func TestPublishNackedWithinTimeout(t *testing.T) {
	client, connection, channel, confirmation := setupMocksForSuccessfulPublish()

	resetCall(confirmation.ExpectedCalls, "Acked", t)
	confirmation.On("Acked").Return(false)

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	err = publisher.Publish(context.Background(), makePublishMessage())

	assert.EqualError(t, err, "received nack from rabbitmq publishing confirmation")
	client.AssertExpectations(t)
	connection.AssertExpectations(t)
	channel.AssertExpectations(t)
	confirmation.AssertExpectations(t)
}

func TestPublishTimeoutBeforeAck(t *testing.T) {
	client, connection, channel, confirmation := setupMocksForSuccessfulPublish()

	resetCall(confirmation.ExpectedCalls, "Done", t)
	resetCall(confirmation.ExpectedCalls, "Acked", t)
	emptyConfirmationChan := make(<-chan struct{})
	confirmation.On("Done").Return(emptyConfirmationChan)

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	err = publisher.Publish(context.Background(), makePublishMessage())

	assert.EqualError(t, err, "timeout waiting for publish confirmation after 20ms")
	client.AssertExpectations(t)
	connection.AssertExpectations(t)
	channel.AssertExpectations(t)
	confirmation.AssertExpectations(t)
}

func TestPublishTwiceReusingSameConnection(t *testing.T) {
	client, connection, channel, confirmation := setupMocksForSuccessfulPublish()

	// Re-use same chan to allow ACKing both publishes
	confirmationChan := make(chan struct{}, 2)
	confirmationChan <- struct{}{}
	confirmationChan <- struct{}{}
	var confirmationChanRet <-chan struct{} = confirmationChan
	resetCall(confirmation.ExpectedCalls, "Done", t)
	confirmation.On("Done").Return(confirmationChanRet)

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	err = publisher.Publish(context.Background(), makePublishMessage())
	require.NoError(t, err)
	err = publisher.Publish(context.Background(), makePublishMessage())
	require.NoError(t, err)

	client.AssertNumberOfCalls(t, "DialConfig", 1)
	client.AssertExpectations(t)
	connection.AssertExpectations(t)
	channel.AssertExpectations(t)
	confirmation.AssertExpectations(t)
}

func TestRestoreUnhealthyConnectionDuringPublish(t *testing.T) {
	client, connection, channel, confirmation := setupMocksForSuccessfulPublish()

	// Capture the channel that the amqp library uses to notify about connection issues so that we can simulate the notification
	resetCall(connection.ExpectedCalls, "NotifyClose", t)
	var connectionErrChan chan *amqp.Error
	connection.On("NotifyClose", mock.Anything).Return(make(chan *amqp.Error)).Run(func(args mock.Arguments) {
		connectionErrChan = args.Get(0).(chan *amqp.Error)
	})

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	connectionErrChan <- amqp.ErrClosed
	connection.On("Close").Return(nil)

	err = publisher.Publish(context.Background(), makePublishMessage())

	require.NoError(t, err)
	client.AssertNumberOfCalls(t, "DialConfig", 2) // Connected twice
	client.AssertExpectations(t)
	connection.AssertExpectations(t)
	connection.AssertNumberOfCalls(t, "Close", 1)
	channel.AssertExpectations(t)
	confirmation.AssertExpectations(t)
}

// Tests code path where connection is closed right after checking the connection error channel
func TestRestoreClosedConnectionDuringPublish(t *testing.T) {
	client, connection, channel, confirmation := setupMocksForSuccessfulPublish()

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	resetCall(connection.ExpectedCalls, "IsClosed", t)
	connection.On("IsClosed").Return(true)

	err = publisher.Publish(context.Background(), makePublishMessage())
	require.NoError(t, err)
	client.AssertNumberOfCalls(t, "DialConfig", 2) // Connected twice
	client.AssertExpectations(t)
	connection.AssertExpectations(t)
	channel.AssertExpectations(t)
	confirmation.AssertExpectations(t)
}

func TestFailRestoreConnectionDuringPublishing(t *testing.T) {
	client, connection, _, _ := setupMocksForSuccessfulPublish()

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)
	client.AssertNumberOfCalls(t, "DialConfig", 1)

	resetCall(connection.ExpectedCalls, "IsClosed", t)
	connection.On("IsClosed").Return(true)

	resetCall(client.ExpectedCalls, "DialConfig", t)
	client.On("DialConfig", connectURL, mock.Anything).Return(nil, errors.New("simulated connection error"))

	err = publisher.Publish(context.Background(), makePublishMessage())
	assert.EqualError(t, err, "failed attempt at restoring unhealthy connection\nsimulated connection error")
	client.AssertNumberOfCalls(t, "DialConfig", 2) // Tried reconnecting
}

func TestErrCreatingChannel(t *testing.T) {
	client, connection, _, _ := setupMocksForSuccessfulPublish()

	resetCall(connection.ExpectedCalls, "Channel", t)
	connection.On("Channel").Return(nil, errors.New("simulated error creating channel"))

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	err = publisher.Publish(context.Background(), makePublishMessage())
	assert.EqualError(t, err, "simulated error creating channel")
}

func TestErrSettingChannelConfirmMode(t *testing.T) {
	client, _, channel, _ := setupMocksForSuccessfulPublish()

	resetCall(channel.ExpectedCalls, "Confirm", t)
	channel.On("Confirm", false).Return(errors.New("simulated error setting channel confirm mode"))

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	err = publisher.Publish(context.Background(), makePublishMessage())
	assert.EqualError(t, err, "simulated error setting channel confirm mode")
}

func TestErrPublishing(t *testing.T) {
	client, connection, _, _ := setupMocksForSuccessfulPublish()

	// resetCall(channel.ExpectedCalls, "PublishWithDeferredConfirmWithContext") doesn't work so need to recreate the mock
	channel := mockChannel{}
	channel.On("Confirm", false).Return(nil)
	channel.On("PublishWithDeferredConfirmWithContext", mock.Anything, exchange, routingKey, true, false, mock.MatchedBy(isPersistentDeliverMode)).Return(nil, errors.New("simulated error publishing"))
	channel.On("Close").Return(nil)
	resetCall(connection.ExpectedCalls, "Channel", t)
	connection.On("Channel").Return(&channel, nil)

	publisher, err := NewConnection(zap.NewNop(), client, makeDialConfig())
	require.NoError(t, err)

	err = publisher.Publish(context.Background(), makePublishMessage())
	assert.EqualError(t, err, "error publishing message\nsimulated error publishing")
}

func setupMocksForSuccessfulPublish() (*mockClient, *mockConnection, *mockChannel, *mockDeferredConfirmation) {
	client := mockClient{}
	connection := mockConnection{}
	channel := mockChannel{}
	confirmation := mockDeferredConfirmation{}

	client.On("DialConfig", mock.Anything, mock.Anything).Return(&connection, nil)
	connection.On("NotifyClose", mock.Anything).Return(make(chan *amqp.Error))
	connection.On("Channel").Return(&channel, nil)
	connection.On("IsClosed").Return(false)

	channel.On("Confirm", false).Return(nil)
	channel.On("PublishWithDeferredConfirmWithContext", mock.Anything, exchange, routingKey, true, false, mock.MatchedBy(isPersistentDeliverMode)).Return(&confirmation, nil)
	channel.On("Close").Return(nil)

	confirmationChan := make(chan struct{}, 1)
	confirmationChan <- struct{}{}
	var confirmationChanRet <-chan struct{} = confirmationChan
	confirmation.On("Done").Return(confirmationChanRet)
	confirmation.On("Acked").Return(true)

	return &client, &connection, &channel, &confirmation
}

func isPersistentDeliverMode(p amqp.Publishing) bool {
	return p.DeliveryMode == amqp.Persistent
}

func resetCall(calls []*mock.Call, methodName string, t *testing.T) {
	for _, call := range calls {
		if call.Method == methodName {
			call.Unset()
			return
		}
	}
	t.FailNow()
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) DialConfig(url string, config amqp.Config) (Connection, error) {
	args := m.Called(url, config)

	if connection := args.Get(0); connection != nil {
		return connection.(Connection), args.Error(1)
	}
	return nil, args.Error(1)
}

type mockConnection struct {
	mock.Mock
}

func (m *mockConnection) IsClosed() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockConnection) Channel() (Channel, error) {
	args := m.Called()
	if channel := args.Get(0); channel != nil {
		return channel.(Channel), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockConnection) NotifyClose(receiver chan *amqp.Error) chan *amqp.Error {
	args := m.Called(receiver)
	return args.Get(0).(chan *amqp.Error)
}

func (m *mockConnection) Close() error {
	args := m.Called()
	return args.Error(0)
}

type mockChannel struct {
	mock.Mock
}

func (m *mockChannel) Confirm(noWait bool) error {
	args := m.Called(noWait)
	return args.Error(0)
}

func (m *mockChannel) PublishWithDeferredConfirmWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) (DeferredConfirmation, error) {
	args := m.Called(ctx, exchange, key, mandatory, immediate, msg)
	if confirmation := args.Get(0); confirmation != nil {
		return confirmation.(DeferredConfirmation), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockChannel) IsClosed() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *mockChannel) Close() error {
	args := m.Called()
	return args.Error(0)
}

type mockDeferredConfirmation struct {
	mock.Mock
}

func (m *mockDeferredConfirmation) Done() <-chan struct{} {
	args := m.Called()
	return args.Get(0).(<-chan struct{})
}

func (m *mockDeferredConfirmation) Acked() bool {
	args := m.Called()
	return args.Bool(0)
}

func makePublishMessage() Message {
	return Message{
		Exchange:   exchange,
		RoutingKey: routingKey,
		Body:       make([]byte, 1),
	}
}

func makeDialConfig() DialConfig {
	return DialConfig{
		URL:                        connectURL,
		PublishConfirmationTimeout: time.Millisecond * 20,
		Durable:                    true,
	}
}
