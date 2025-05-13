package main

import (
	"errors"
	"strings"
	"testing"
)

// TestAppCloseScenarios tests various scenarios for the Close method
func TestAppCloseScenarios(t *testing.T) {
	tests := map[string]struct {
		setupFunc     func() *App
		expectError   bool
		validateErr   func(t *testing.T, err error)
		validateState func(t *testing.T, app *App)
	}{
		"NilLogger": {
			setupFunc: func() *App {
				app := NewTestApp()
				app.Logger = nil
				return app
			},
			expectError: false,
		},
		"NilLocker": {
			setupFunc: func() *App {
				app := NewTestApp()
				mockLogger := &MockLogger{}
				app = WithMockLogger(app, mockLogger)
				app.Locker = nil
				return app
			},
			expectError: false,
			validateState: func(t *testing.T, app *App) {
				mockLogger := app.Logger.(*MockLogger)
				if !mockLogger.CloseCalled {
					t.Error("Expected logger.Close to be called even when locker is nil")
				}
			},
		},
		"LoggerError": {
			setupFunc: func() *App {
				app := NewTestApp()
				expectedErr := errors.New("mock logger close error")
				mockLogger := &MockLogger{
					CloseErr: expectedErr,
				}
				app = WithMockLogger(app, mockLogger)
				return app
			},
			expectError: true,
			validateErr: func(t *testing.T, err error) {
				expectedErr := "mock logger close error"
				if err == nil || !strings.Contains(err.Error(), expectedErr) {
					t.Errorf("Expected error with '%s', got: %v", expectedErr, err)
				}
			},
		},
		"LockerError": {
			setupFunc: func() *App {
				app := NewTestApp()
				mockLogger := &MockLogger{}
				app = WithMockLogger(app, mockLogger)

				expectedErr := errors.New("mock locker release error")
				mockLocker := &MockLocker{
					ReleaseErr: expectedErr,
				}
				app = WithMockLocker(app, mockLocker)
				return app
			},
			expectError: true,
			validateErr: func(t *testing.T, err error) {
				expectedErr := "mock locker release error"
				if err == nil || !strings.Contains(err.Error(), expectedErr) {
					t.Errorf("Expected error with '%s', got: %v", expectedErr, err)
				}
			},
			validateState: func(t *testing.T, app *App) {
				mockLogger := app.Logger.(*MockLogger)
				mockLocker := app.Locker.(*MockLocker)

				if !mockLogger.CloseCalled {
					t.Error("Expected logger.Close to be called")
				}

				if !mockLocker.ReleaseCalled {
					t.Error("Expected locker.Release to be called")
				}
			},
		},
		"BothErrors": {
			setupFunc: func() *App {
				app := NewTestApp()

				loggerErr := errors.New("mock logger close error")
				mockLogger := &MockLogger{
					CloseErr: loggerErr,
				}
				app = WithMockLogger(app, mockLogger)

				lockerErr := errors.New("mock locker release error")
				mockLocker := &MockLocker{
					ReleaseErr: lockerErr,
				}
				app = WithMockLocker(app, mockLocker)
				return app
			},
			expectError: true,
			validateErr: func(t *testing.T, err error) {
				errStr := err.Error()
				loggerErrMsg := "mock logger close error"
				lockerErrMsg := "mock locker release error"

				if !strings.Contains(errStr, loggerErrMsg) {
					t.Errorf("Expected error to contain logger error message '%s', got: %v", loggerErrMsg, errStr)
				}

				if !strings.Contains(errStr, lockerErrMsg) {
					t.Errorf("Expected error to contain locker error message '%s', got: %v", lockerErrMsg, errStr)
				}
			},
			validateState: func(t *testing.T, app *App) {
				mockLogger := app.Logger.(*MockLogger)
				mockLocker := app.Locker.(*MockLocker)

				if !mockLogger.CloseCalled {
					t.Error("Expected logger.Close to be called")
				}

				if !mockLocker.ReleaseCalled {
					t.Error("Expected locker.Release to be called")
				}
			},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			app := test.setupFunc()
			err := app.Close()

			if test.expectError && err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !test.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if err != nil && test.validateErr != nil {
				test.validateErr(t, err)
			}

			if test.validateState != nil {
				test.validateState(t, app)
			}
		})
	}
}
