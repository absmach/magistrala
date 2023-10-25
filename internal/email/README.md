# Magistrala Email Agent

Magistrala Email Agent is used for sending emails. It wraps basic SMTP features and 
provides a simple API that Magistrala services can use to send email notifications.

## Configuration

Magistrala Email Agent is configured using the following configuration parameters:

| Parameter                           | Description                                                             |
| ----------------------------------- | ----------------------------------------------------------------------- |
| MG_EMAIL_HOST                       | Mail server host                                                        |
| MG_EMAIL_PORT                       | Mail server port                                                        |
| MG_EMAIL_USERNAME                   | Mail server username                                                    |
| MG_EMAIL_PASSWORD                   | Mail server password                                                    |
| MG_EMAIL_FROM_ADDRESS               | Email "from" address                                                    |
| MG_EMAIL_FROM_NAME                  | Email "from" name                                                       |
| MG_EMAIL_TEMPLATE                   | Email template for sending notification emails                          |

There are two authentication methods supported: Basic Auth and CRAM-MD5.
If `MG_EMAIL_USERNAME` is empty, no authentication will be used.
