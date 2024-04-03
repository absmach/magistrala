# Magistrala Users Groups Things and Channels E2E Testing Tool

A simple utility to create a list of groups and users connected to these groups and channels and things connected to these channels.

## Installation

```bash
cd tools/e2e
make
```

### Usage

```bash
./e2e --help
Tool for testing end-to-end flow of Magistrala by doing a couple of operations namely:
1. Creating, viewing, updating and changing status of users, groups, things and channels.
2. Connecting users and groups to each other and things and channels to each other.
3. Sending messages from things to channels on all 4 protocol adapters (HTTP, WS, CoAP and MQTT).
Complete documentation is available at https://docs.magistrala.abstractmachines.fr


Usage:

  e2e [flags]


Examples:

Here is a simple example of using e2e tool.
Use the following commands from the root Magistrala directory:

go run tools/e2e/cmd/main.go
go run tools/e2e/cmd/main.go --host 142.93.118.47
go run tools/e2e/cmd/main.go --host localhost --num 10 --num_of_messages 100 --prefix e2e


Flags:

  -h, --help                   help for e2e
  -H, --host string            address for a running Magistrala instance (default "localhost")
  -n, --num uint               number of users, groups, channels and things to create and connect (default 10)
  -N, --num_of_messages uint   number of messages to send (default 10)
  -p, --prefix string          name prefix for users, groups, things and channels
```

To use `-H` option, you can specify the address for the Magistrala instance as an argument when running the program. For example, if the Magistrala instance is running on another computer with the IP address 192.168.0.1, you could use the following command:

```bash
go run tools/e2e/cmd/main.go --host 142.93.118.47
```

This will tell the program to connect to the Magistrala instance running on the specified IP address.

If you want to create a list of channels with certificates:

```bash
go run tools/e2e/cmd/main.go --host localhost --num 10 --num_of_messages 100 --prefix e2e
```

Example of output:

```bash
created user with token eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE2ODEyMDYwMjMsImlhdCI6MTY4MTIwNTEyMywiaWRlbnRpdHkiOiJlMmUtbGF0ZS1zaWxlbmNlQGVtYWlsLmNvbSIsImlzcyI6ImNsaWVudHMuYXV0aCIsInN1YiI6IjdlZDIyY2IyLTRlMzQtNDhiZi04Y2RlLTIxMjZiYzYyYzY4MyIsInR5cGUiOiJhY2Nlc3MifQ.AdExNYs5mVQNpo_ejJDq7KTC5dKkZWmgM9FJvTM2T_GM2LE9ASQv0ymC4wS3PDXKWf-OcaR8DJIxE6WiG3fztQ
created users of ids:
9e87bc1d-0889-4252-a3df-36e02edfc859
c1e4901a-fb7f-45e9-b934-c55194b1d028
c341a9cb-542b-4c3b-afd6-c98e04ed5e7e
8cfc886b-21fa-4205-80b4-3601827b94ff
334984d7-30eb-4b06-92b8-5ec182bebac5
created groups of ids:
7744ec55-c767-4137-be96-0d79699772a4
c8fe4d9d-3ad6-4687-83c0-171356f3e4f6
513f7295-0923-4e21-b41a-3cfd1cb7b9b9
54bd71ea-3c22-401e-89ea-d58162b983c0
ae91b327-4c40-4e68-91fe-cd6223ee4e99
created things of ids:
5909a907-7413-47d4-b793-e1eb36988a5f
f9b6bc18-1862-4a24-8973-adde11cb3303
c2bd6eed-6f38-464c-989c-fe8ec8c084ba
8c76702c-0534-4246-8ed7-21816b4f91cf
25005ca8-e886-465f-9cd1-4f3c4a95c6c1
created channels of ids:
ebb0e5f3-2241-4770-a7cc-f4bbd06134ca
d654948d-d6c1-4eae-b69a-29c853282c3d
2c2a5496-89cf-47e6-9d38-5fd5542337bd
7ab3319d-269c-4b07-9dc5-f9906693e894
5d8fa139-10e7-4683-94f3-4e881b4db041
created policies for users, groups, things and channels
viewed users, groups, things and channels
updated users, groups, things and channels
sent messages to channels
```
