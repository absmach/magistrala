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
| MF_EMAIL_PASSWORD                   | Mail server password for Basic authentication                           |
| MF_EMAIL_SECRET                     | Mail server secret for CRAM-MD5 authentication                          |
| MF_EMAIL_FROM_ADDRESS               | Email "from" address                                                    |
| MF_EMAIL_FROM_NAME                  | Email "from" name                                                       |
| MF_EMAIL_TEMPLATE                   | Email template for sending notification emails                          |

There are two authentication methods supported: Basic Auth and CRAM-MD5.
`MF_EMAIL_SECRET` indicates that `CRAM-MD5` authentication will be used.
`MF_EMAIL_PASSWORD` indicates that `Basic` authentication will be used.
If both `MF_EMAIL_SECRET` and `MF_EMAIL_PASSWORD` are present, `CRAM-MD5` authentication will be used.
If `MF_EMAIL_USERNAME` is empty or both `MF_EMAIL_SECRET` and `MF_EMAIL_PASSWORD` are empty, 
no authentication will be used.