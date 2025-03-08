# TodoISSH Test Suite

This directory contains tests for the TodoISSH application. The test suite is designed to ensure the application works correctly and reliably, focusing on both individual components (unit tests) and their interactions (integration tests).

## Test Organization

The tests are organized as follows:

- **Unit Tests**: Located in each package directory with the `_test.go` suffix
  - `pkg/todo/todo_test.go`: Tests for the todo store functionality
  - `pkg/user/user_test.go`: Tests for the user management functionality (authentication, registration)
  - `pkg/ssh/ssh_test.go`: Tests for the SSH server functionality

- **Integration Tests**: Located in the `test/integration` directory
  - `integration_test.go`: Tests interactions between multiple components

## Test Categories

### Unit Tests

The unit tests are focused on testing individual components in isolation, ensuring that each package functions correctly on its own. Key areas tested include:

1. **Basic Functionality Tests**
   - CRUD operations
   - Error handling
   - Edge cases

2. **File System Operations**
   - Directory creation
   - File read/write
   - Permissions

3. **Concurrent Operations**
   - Thread safety
   - Race conditions

4. **Data Persistence**
   - Saving and loading data
   - Data integrity

### Integration Tests

The integration tests focus on testing how different components interact with each other. Key areas tested include:

1. **User and Todo Store Integration**
   - User registration and authentication
   - Todo operations across users
   - Data isolation

2. **Edge Cases**
   - User registration edge cases
   - Todo operations edge cases

3. **Concurrent User Operations**
   - Multiple users operating simultaneously
   - Thread safety across components

4. **Data Persistence Across Components**
   - State preservation across restarts
   - Data integrity across components

5. **Error Recovery**
   - Handling corrupted data
   - System resilience

## Running the Tests

### Running All Tests

To run all tests in the project:

```sh
go test ./...
```

### Running Unit Tests

To run unit tests for a specific package:

```sh
go test ./pkg/todo    # Run todo package tests
go test ./pkg/user    # Run user package tests
go test ./pkg/ssh     # Run SSH server tests
```

### Running Integration Tests

To run integration tests:

```sh
go test ./test/integration
```

### Running with Verbose Output

For more detailed output, add the `-v` flag:

```sh
go test -v ./...
```

### Running Specific Tests

To run a specific test:

```sh
go test -v ./pkg/todo -run TestAddWithMock
go test -v ./test/integration -run TestUserDataIsolation
```

## Test Coverage

To run tests with coverage information:

```sh
go test -cover ./...
```

For a detailed coverage report:

```sh
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Test Design Principles

1. **Isolation**: Unit tests should test a single component in isolation.
2. **Independence**: Tests should be independent of each other and not rely on state from previous tests.
3. **Determinism**: Tests should be deterministic and not depend on timing or external factors.
4. **Completeness**: Tests should cover normal cases, edge cases, and error conditions.
5. **Performance**: Tests should be fast enough to run frequently during development.
6. **Readability**: Tests should be clear and easy to understand, serving as documentation.

## Mocking and Test Helpers

The test suite includes several helper functions and mock implementations to facilitate testing:

- `setupTestStore()`: Creates a temporary directory and initialized store for testing
- `cleanupTestStore()`: Cleans up temporary test directories
- `setupTestEnvironment()`: Sets up a complete test environment with user and todo stores

## Continuous Integration

These tests are designed to be run as part of a CI pipeline to ensure that changes to the codebase don't introduce regressions. 