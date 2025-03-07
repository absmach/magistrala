# SuperMQ Email Agent

SuperMQ Email Agent is used for sending emails. It wraps basic SMTP features and 
provides a simple API that SuperMQ services can use to send email notifications.

## Configuration

SuperMQ Email Agent is configured using the following configuration parameters:

| Parameter                           | Description                                                             |
| ----------------------------------- | ----------------------------------------------------------------------- |
| SMQ_EMAIL_HOST                       | Mail server host                                                        |
| SMQ_EMAIL_PORT                       | Mail server port                                                        |
| SMQ_EMAIL_USERNAME                   | Mail server username                                                    |
| SMQ_EMAIL_PASSWORD                   | Mail server password                                                    |
| SMQ_EMAIL_FROM_ADDRESS               | Email "from" address                                                    |
| SMQ_EMAIL_FROM_NAME                  | Email "from" name                                                       |
| SMQ_EMAIL_TEMPLATE                   | Email template for sending notification emails                          |

There are two authentication methods supported: Basic Auth and CRAM-MD5.
If `SMQ_EMAIL_USERNAME` is empty, no authentication will be used.
