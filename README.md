# mailout - CaddyServer SMTP Client

Post form data from a website to this route and receive the data as nicely formatted email.
  
Caddy config options:

```
mailout endpoint {
    keybaseio       https://keybase.io/cyrill/key.asc
	public_key      path/to/pgp.pub
	logdir          path/to/logdir
	
	success_uri     http://mydomain.com/email_sent_confirmation.html    
	
	to              recipient_to@domain.email        
	cc              recipient_cc1@domain.email, recipient_cc2@domain.email        
	bcc             recipient_bcc1@domain.email, recipient_bcc2@domain.email
    subject         Email from {{.firstname}} {{.lastname}}
	body            path/to/tpl.[txt|html]
	
	username        [ENV:MY_SMTP_USERNAME|gopher]
	password        [ENV:MY_SMTP_PASSWORD|g0ph3r]
	host            [ENV:MY_SMTP_HOST|smtp.gmail.com]
	port            [ENV:MY_SMTP_PORT|25|587|465]
}
```

### Email template

- Plain text template language [https://golang.org/pkg/text/template/](https://golang.org/pkg/text/template/).
- HTML template language [https://golang.org/pkg/html/template/](https://golang.org/pkg/html/template/).

An email template for an outgoing mail may look like in plain text:

```
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
