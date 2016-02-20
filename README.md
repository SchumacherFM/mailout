# mailout - CaddyServer SMTP Client

Post form data from a website to this route and receive the data as nicely formatted email.
  
Caddy config options:

```
mailout [endpoint] {
	publickey      [path/to/pgp.pub|ENV:MY_PGP_KEY_PATH|https://keybase.io/cyrill/key.asc]
	maillog         [path/to/logdir|or empty]
		
	to              recipient_to@domain.email       
	cc              "recipient_cc1@domain.email, recipient_cc2@domain.email"        
	bcc             "recipient_bcc1@domain.email, recipient_bcc2@domain.email"
    subject         "Email from {{.firstname}} {{.lastname}}"
	body            path/to/tpl.[txt|html]
	
	username        "[ENV:MY_SMTP_USERNAME|gopher]"
	password        "[ENV:MY_SMTP_PASSWORD|g0ph3r]"
	host            "[ENV:MY_SMTP_HOST|smtp.gmail.com]"
	port            [ENV:MY_SMTP_PORT|25|465|587]
}
```

- endpoint: Can be any path but your POST request must match it. Default path: `/mailout`
- publickey: if provided mails get encrypted. Set a path to a file, an environment variable or an URL to a key on a HTTPS site.
- maillog: Specify a directory, which gets created recursively, and emails plus errors gets logged in there. Leaving the maillog setting empty does not log anything. Every sent email is saved into its own file. Strict file permissions apply. 
- to, cc, bcc: Multiple email addresses must be separated by a colon and within double quotes.
- subject: Has the same functionality as the body template.
- body: Text or HTML template stored on the hard disk of your server. More details below.
- username, password, host: Self explanatory.
- port: Plain text on port 25, SSL use port 465, for TLS use port 587. Internally for TLS the host name gets verified with the certificate.

The default filename for an encrypted message attached to an email is: *encrypted.gpg*. The extension
`.gpg` has been chosen to allow easy handling with [https://www.gnupg.org/](https://www.gnupg.org/)
If you don't like this file name you can overwrite it with the key `publickeyAttachmentFileName`.

To implement a fully working *This is an OpenPGP/MIME encrypted message (RFC 4880 and 3156)* PGP attachment, 
I need some help. It's possible that the gomail package needs to be refactored.

### HTML form, JSON and email template

The rendering engine for the email templates depends on the suffix of the template file name. 

- `.txt`: Plain text template language [https://golang.org/pkg/text/template/](https://golang.org/pkg/text/template/).
- `.html`: HTML template language [https://golang.org/pkg/html/template/](https://golang.org/pkg/html/template/).

Create a simple HTML form with some JavaScript and AJAX functions.

Mandatory input field is `email`. Optional recommended field: `name`. Those two fields
will be later joined to create the `From:` header of an email.

The following snipped has been extracted from a [Hugo](http://gohugo.io) template.

```
  <div id="contactThankYou" style="display:hidden;">Thank you for contacting us!</div>
  <form action="#" id="myContactForm" method="POST">
    <div class="row uniform 50%">
      <div class="6u 12u$(xsmall)">
        <input type="text" name="name" id="name"
               placeholder="{{ .Site.Params.contact.form.name }}" required/>
      </div>
      <div class="6u$ 12u$(xsmall)">
        <input type="email" name="email" id="email"
               placeholder="{{ .Site.Params.contact.form.email }}" required/>
      </div>
      <div class="12u$">
        <textarea name="message" id="message" placeholder="{{ .Site.Params.contact.form.message }}"
                  rows="4" required></textarea>
      </div>
      <input type="hidden" name="user_agent" value="Will be filled our via JavaScript"/>
      <ul class="actions">
        <li><input type="submit" value="{{ .Site.Params.contact.form.submit }}"/></li>
      </ul>
    </div>
  </form>
```

Server response on success (Status 200):

```
{}
```

Server response on error (Status 200):

```
{"error":"Invalid email address: \"doe.john_AT_nonexistantServer.email\""}
```

Non 200 Statuses are returned as non-json strings.

A jQuery AJAX handler might look like (untested):

```
$(document).ready(function() {

    $('#myContactForm').submit(function(event) {

        $.ajax({
            type        : 'POST', 
            url         : 'https://myCaddyServer.com/mailout', 
            data        : $('#myContactForm').serialize(),
            dataType    : 'json',
            encode      : true
        })
        .done(function(data) {

            console.log(data); 
            $('#contactThankYou').show();
            $('#myContactForm').hide();

        })
         .fail(function() {
            alert( "error" );
         });

        event.preventDefault();
    });

});    
```

An email template for an outgoing mail may look like in plain text:

```
Hello,

please find below a new contact:

Name            {{.Form.Get "name"}}
Email           {{.Form.Get "email"}}

Message:
{{.Form.Get "message"}}

User Agent: {{.Form.Get "user_agent"}}
```

### GMail

If you use Gmail as outgoing server these pages can help:

- [Google SMTP settings to send mail from a printer, scanner, or app](https://support.google.com/a/answer/176600)
- [Allowing less secure apps to access your account](https://support.google.com/accounts/answer/6010255)

I need some help to pimp the authentication feature of [gomail](https://github.com/go-gomail/gomail/blob/master/smtp.go#L41) to avoid switching on the less secure "feature". 

# Todo

- file uploads
- implement ideas and improvements in open issues

# Contribute

Send me a pull request or open an issue if you encounter a bug or something can be improved!

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
