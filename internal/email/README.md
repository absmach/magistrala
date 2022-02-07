# Mainflux Email Agent

Mainflux Email Agent is used for sending emails. It wraps basic SMTP features and 
provides a simple API that Mainflux services can use to send email notifications.

## Configuration

Mainflux Email Agent is configured using the following configuration parameters:

| Parameter                           | Description                                                             |
| ----------------------------------- | ----------------------------------------------------------------------- |
| MF_EMAIL_HOST                       | Mail server host                                                        |
| MF_EMAIL_PORT                       | Mail server port                                                        |
| MF_EMAIL_USERNAME                   | Mail server username                                                    |
| MF_EMAIL_PASSWORD                   | Mail server password                                                    |
| MF_EMAIL_FROM_ADDRESS               | Email "from" address                                                    |
| MF_EMAIL_FROM_NAME                  | Email "from" name                                                       |
| MF_EMAIL_TEMPLATE                   | Email template for sending notification emails                          |

There are two authentication methods supported: Basic Auth and CRAM-MD5.
If `MF_EMAIL_USERNAME` is empty, no authentication will be used.
