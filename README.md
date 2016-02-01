# mailout

Post form data from a website to this microservice and receive the data as nicely formatted email.
  
Secured via JSON Web Tokens (JWT), rate limit and other "fancy" checks like referer, IP, host (cough, cough).

## Use cases

- Security aware user who would not like to use a 3rd party cloud hosted services 
- Static site generators like [Hugo](https://gohugo.io) can have their e.g. contact form
- eCommerce stores can outsource email sending and generating

## General features

- One binary
- Runs on nearly every operating system
- Capable of up to 20m requests per hour (not on a R-Pi)
- HTTP2 build in

## Frontend

1. Query the mailout microservice to receive a JWT for each user or visitor
2. Embed the JWT into your form
3. User posts the form data plus the JWT back to the mailout microservice
4. Email will be generated and send out
5. Email may get logged

## Backend

- Write {text,html} templates based on the real email design and fields.
- XSS Secure HTML emails (except you make them unsecure ;-) ).
- Configure the checks for rate limit, IP, referer, host ...
- Optional PGP encrypted emails
- Limitation of POST data size per defined email
- Attachments from POST data will be appended
- Use of internal *nix email MTAs (mail transfer agents)
- Use of external MTA services via SMTP/TLS
- Retry for external MTA services or "load balancing" to other MTAs
- TLS to receive the POST data

### Email template

- Plain text template language [https://golang.org/pkg/text/template/](https://golang.org/pkg/text/template/).
- HTML template language [https://golang.org/pkg/html/template/](https://golang.org/pkg/html/template/).

An email template for an outgoing mail may look like in plain text:

```
PGP: path/to/my/public.key <== optional!
From: {{.Name}} <{{.Email}}>
To: reception@hotel-california.travel
Subject: Booking request from webpage (IP: {{.RemoteAddr}})

Hello,

please find below a new request for a booking:

Name            {{.Name}}
Email           {{.Email}}
Arrival time    {{.ATime}}
Departure time  {{.DTime}}

Message:
{{.Message}}

{{.RequestServerInformations}}
```

## Future fancy features

- [gRPC](http://www.grpc.io/)
- Backend HTML user interface
    - Create, edit and delete templates
    - Create, edit and delete outgoing servers
    - PGP handling
- [Let's encrypt](https://letsencrypt.org/) integration
- Persisted email log and viewable for UI.
- Template reloading on-the-fly or watched for changes.
- etc

# Contribute

Send me a pull request or open an issue!

Multi-time pull request senders gets collaborator access.

# License

[Cyrill Schumacher](https://github.com/SchumacherFM) - [My pgp public key](https://www.schumacher.fm/cyrill.asc)

Copyright 2016 Cyrill Schumacher All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
