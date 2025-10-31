# Magistrala Rules Engine

The Magistrala Rules Engine (RE) is a service that enables real-time message processing and transformation through user-defined rules. It allows you to create rules that process incoming messages using Lua scripts and publish the results to output channels.

## Architecture

The Rules Engine operates by:
1. Listening for messages on configured input channels
2. Processing these messages through Lua scripts
3. Optionally publishing results to output channels
4. Supporting scheduled rule execution based on various recurring patterns

## Core Concepts

### Rules

A rule consists of:

- `id` - Unique identifier
- `name` - Human-readable name
- `domain` - Domain the rule belongs to
- `input_channel` - Channel to listen for incoming messages
- `input_topic` - Specific topic within the input channel
- `logic` - Lua script that processes the message
- `output_channel` - (Optional) Channel to publish results to
- `output_topic` - (Optional) Topic within the output channel
- `schedule` - (Optional) Scheduling configuration
- `status` - Rule state (enabled/disabled/deleted)
- `metadata` - Additional rule metadata

A rule can be in one of these states:
- `enabled` - Rule is active and processing messages
- `disabled` - Rule is inactive and won't process messages
- `deleted` - Rule is marked for deletion

### Message Processing

When a message arrives on a rule's input channel, the Rules Engine:

1. Creates a Lua environment
2. Injects the message as a global variable with the following structure:
   ```lua
   message = {
     channel = "channel_name",
     subtopic = "subtopic_name",
     publisher = "publisher_id",
     protocol = "protocol_name",
     created = timestamp,
     payload = [byte_array]
   }
   ```
3. Executes the rule's Lua script
4. If the script returns a non-nil value and an output channel is configured, publishes the result

### Scheduling

Rules can be scheduled to run at specific times with various recurring patterns. The scheduler works through several key components:

#### Schedule Structure
```go
type Schedule struct {
    StartDateTime   time.Time  // When the schedule becomes active
    Time            time.Time  // Specific time for the rule to run
    Recurring       Recurring  // None, Daily, Weekly, Monthly
    RecurringPeriod uint      // Interval between executions: 1 = every interval, 2 = every second interval, etc.
}
```

#### How Scheduling Works

1. **Initialization**:
   - The scheduler starts when the service begins running via `StartScheduler()`
   - It uses a ticker to check for rules that need to be executed at regular intervals

2. **Rule Evaluation**:
   - For each tick, the scheduler:
     - Gets all enabled rules scheduled before the current time
     - For each rule, checks if it should run using `shouldRunRule()`
     - If a rule should run, processes it asynchronously

3. **Execution Timing**:
   The `shouldRunRule()` function determines if a rule should run by checking:
   - If the rule's start time has been reached
   - If the current time matches the scheduled execution time
   - For recurring rules:
     - **Daily**: Checks if the correct number of days have passed since start
     - **Weekly**: Checks if the correct number of weeks have passed since start
     - **Monthly**: Checks if the correct number of months have passed since start

4. **Recurring Patterns**:
   - `None`: Rule runs once at the specified time
   - `Daily`: Rule runs every N days where N is the RecurringPeriod
   - `Weekly`: Rule runs every N weeks
   - `Monthly`: Rule runs every N months

For example, to run a rule:
- Every day at 9 AM: Set recurring to "daily" with recurring_period = 1
- Every other week: Set recurring to "weekly" with recurring_period = 2
- Monthly on the 1st: Set recurring to "monthly" with recurring_period = 1

## API Operations

The Rules Engine service provides the following operations:

- `AddRule` - Create a new rule
- `ViewRule` - Retrieve a specific rule
- `UpdateRule` - Modify an existing rule
- `ListRules` - Query rules with filtering options
- `RemoveRule` - Delete a rule
- `EnableRule` - Activate a rule
- `DisableRule` - Deactivate a rule

## Using the API

### Adding a Rule

You can create a new rule using the Rules Engine API. Here's an example using curl:

```bash
curl --location 'http://localhost:9008/8353542f-d8f1-4dce-b787-4af3712f117e/rules' \
--header 'Content-Type: application/json' \
--header 'Authorization: Bearer <access_token>' \
--data '{
  "name": "High Temperature Alert",
  "input_channel": "sensors",
  "input_topic": "temperature",
  "logic": {
    "type": 0,
    "value": "if message.payload > 30 then return '\''Temperature too high!'\'' end"
  },
  "output_channel": "alerts",
  "output_topic": "temperature",
  "schedule": {
    "start_datetime": "2024-01-01T00:00",
    "time": "2024-01-01T09:00",
    "recurring": "daily",
    "recurring_period": 1
  }
}'
```

This request:
- Creates a temperature monitoring rule
- Processes messages from the "sensors" channel
- Checks for temperatures above 30 degrees
- Publishes alerts to the "alerts" channel
- Runs daily at 9 AM

The API endpoint follows the format: `http://localhost:9008/{domain_id}/rules`

Required headers:
- `Content-Type: application/json` - Specifies the request body format
- `Authorization: Bearer <access_token>` - Your authentication token

### Example Rule Structure

Here's a breakdown of the rule structure:

```json
{
  "name": "High Temperature Alert",
  "input_channel": "sensors",
  "input_topic": "temperature",
  "logic": {
    "type": 0,
    "value": "if message.payload > 30 then return 'Temperature too high!' end"
  },
  "output_channel": "alerts",
  "output_topic": "temperature",
  "schedule": {
    "start_datetime": "2024-01-01T00:00",
    "time": "2024-01-01T09:00",
    "recurring": "daily",
    "recurring_period": 1
  }
}
```

This rule:
1. Listens on the "sensors" channel, "temperature" topic
2. Checks if temperature exceeds 30 degrees
3. If true, publishes an alert message
4. Runs daily at 9 AM

## Running the Service

To start the Rules Engine service, run:

```bash
make run_addons re
```

This command starts the Rules Engine service using Docker Compose with the configuration defined in [docker-compose.yaml][compose].

## For More Information

- [Magistrala Documentation][doc]
- [Docker Compose][compose]

[doc]: https://docs.magistrala.absmach.eu
[compose]: ../docker/docker-compose.yaml
