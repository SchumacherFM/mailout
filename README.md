# mailout - CaddyServer SMTP Client with PGP 


Post form data from a website to this route and receive the data as nicely
formatted email.

Supports Caddy >= v0.9

Read more: [https://cyrillschumacher.com/projects/2016-02-26-mailout-caddyserver-email-smtp/](https://cyrillschumacher.com/projects/2016-02-26-mailout-caddyserver-email-smtp/)

### Mailout config options in the Caddyfile:

```
mailout [endpoint] {
	maillog         [path/to/logdir|stdout|stderr]
	errorlog        [path/to/logdir|stdout|stderr]

	to              email@address1.tld       
	[cc             "email@address2.tld, email@addressN.tld"]        
	[bcc            "email@addressN.tld, email@addressN.tld"]
    subject         "Email from {{.firstname}} {{.lastname}}"
	body            path/to/tpl.[txt|html]
	[from_email     optional.senders@email.address]
	[from_name      "Optional Senders Name"]

    [email@address1.tld     path/to/pgp1.pub|ENV:MY_PGP_KEY_PATH_1|https://keybase.io/cyrill1/key.asc]
	[email@address2.tld     path/to/pgp2.pub|ENV:MY_PGP_KEY_PATH_2|https://keybase.io/cyrill2/key.asc]
	[email@addressN.tld     path/to/pgpN.pub|ENV:MY_PGP_KEY_PATH_N|https://keybase.io/cyrillN/key.asc]

	username        "ENV:MY_SMTP_USERNAME|gopher"
	password        "ENV:MY_SMTP_PASSWORD|g0ph3r"
	host            "ENV:MY_SMTP_HOST|smtp.gmail.com"
	port             ENV:MY_SMTP_PORT|25|465|587
	
	[ratelimit_interval 24h]
	[ratelimit_capacity 1000]
	
	[skip_tls_verify]
	
	[captcha]
	
	[recaptcha]
	recaptcha_secret    [reCAPTCHA Secret key of your site]
}
```

Configuration values in brackets are optional.

- `endpoint`: Can be any path but your POST request must match it. Default path:
`/mailout`
- [email-address]: if provided mails get encrypted. Set a path to a file, an
environment variable or an URL to a key on a HTTPS site. Key = email address;
value = PGP Key
- `maillog`: Specify a directory, which gets created recursively, and emails will
be written in there, as a backup. Leaving the maillog setting empty does not log
anything. Every sent email is saved into its own file. Strict file permissions
apply. If set to the value "stderr" or "stdout" (without the quotations), then
the output will forwarded to those file descriptors.
- `errorlog`: Specify a directory, which gets created recursively, and errors gets
logged in there. Leaving the errorlog setting empty does not log anything.
Strict file permissions apply. If set to the value "stderr" or "stdout" (without
the quotations), then the output will forwarded to those file descriptors.
- `to`, `cc`, `bcc`: Multiple email addresses must be separated by a colon and within
double quotes.
- `subject`: Has the same functionality as the body template, but text only.
- `body`: Text or HTML template stored on the hard disk of your server. More
details below.
- `from_email`: Email address of the sender, otherwise the email address of the
HTML form from the front end gets used.
- `from_name`: Name of the sender. If empty the email address in the field
`from_email` gets used.
- `username`, `password`, `host`: Self explanatory, access credentials to the SMTP
server.
- `port`: Plain text on port 25, SSL uses port 465, for TLS use port 587.
Internally for TLS the host name gets verified with the certificate of the SMTP
server.
- `ratelimit_interval`: the duration in which the capacity can be consumed. A
duration string is a possibly signed sequence of decimal numbers, each with
optional fraction and a unit suffix, such as "300ms", "1.5h" or "2h45m". Valid
time units are "ns", "us" (or "µs"), "ms", "s", "m", "h". Default: 24h
- `ratelimit_capacity`: the overall capacity within the interval. Default: 1000
- `skip_tls_verify` if added skips the TLS verification process otherwise
hostnames must match.

The default filename for an encrypted message attached to an email is:
*encrypted.gpg*.

The extension `.gpg` has been chosen to allow easy handling with
[https://www.gnupg.org/](https://www.gnupg.org/)

If you don't like this file name you can overwrite it with the key
`publickeyAttachmentFileName`.

To implement a fully working *This is an OpenPGP/MIME encrypted message (RFC
4880 and 3156)* PGP attachment, I need some help. It's possible that the gomail
package needs to be refactored.

Note on sensitive information leakage when using PGP with multiple email message
receivers: For each email address in the to, cc and bcc field you must add a
public PGP key, if not, emails to recipients without a public key won't be
encrypted. For all email addresses with a PGP key, the mailout middleware will
send a separated email encrypted with the key of the receiver.

Rate limit: Does not require external storage since it uses an algorithm called
[Token Bucket](http://en.wikipedia.org/wiki/Token_bucket) [(Go library:
juju/ratelimit)](https://github.com/juju/ratelimit).

**Note**: Current architecture of the mailout pluging allows to set only one
*endpoint and its configuration per virtual host. If you need more endpoints,
*for different email receivers, you must create additional virtual hosts in
*Caddy.

### JSON API

Server response on success (Status 200 OK):

```
{"code":200}
```

Server response on error (Status 422 Unprocessable Entity):

```
{"code":422,"error":"Invalid email address: \"doe.john40nonexistantServer.email\""}
```

Server response on non-POST requests (Status 405 Method Not Allowed):

```
{"code":405,"error":"Method Not Allowed"}
```

Server response on form parse error (Status 400 Bad Request):

```
{"code":400,"error":"Bad request"}
```

Server response on reaching the rate limit (Status 429 Too Many Requests):

```
{"code":429,"error":"Too Many Requests"}
```

Server response on internal errors:
 
```
500 Internal Server Error
```


### Captcha

Example:

add to config: captcha
```html
<div class="form-group text-center">
 <img id="captcha" src="/mailout/captcha">
 <input type="text" id="captcha_text" name="captcha_text" class="form-control" placeholder="Captcha text" required>
</div>
```

After sending the request:
```js
var d = new Date();
$("#captcha").attr("src", "/mailout/captcha?" + d.getTime());
```
https://github.com/steambap/captcha

https://github.com/quasoft/memstore

### ReCaptcha
Example:

add to config: recaptcha and recaptcha_secret
```
recaptcha
recaptcha_secret  6LdnR1QUAAAAAIdxxxxxxxxxxxxx
```
#### [Demo recaptcha + captcha](https://dev.avv.ovh/mailout-test/)


### Email template

The rendering engine for the email templates depends on the suffix of the
template file name.

- `.txt`: Plain text template language
[https://golang.org/pkg/text/template/](https://golang.org/pkg/text/template/).
- `.html`: HTML template language
[https://golang.org/pkg/html/template/](https://golang.org/pkg/html/template/).

### HTML form

Create a simple HTML form with some JavaScript and AJAX functions.

Mandatory input field is `email`. Optional recommended field: `name`. Those two
fields will be later joined to create the `From:` header of an email.

The following snipped has been extracted from a [Hugo](http://gohugo.io)
template.

```html
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
      <input type="hidden" name="user_agent" value="Will be filled out via JavaScript"/>
      <ul class="actions">
        <li><input type="submit" value="{{ .Site.Params.contact.form.submit }}"/></li>
      </ul>
    </div>
  </form>
```

A jQuery AJAX handler might look like:

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

### HTML Form Fields

A must-have form field is the email address: `<input type="text" name="email" value=""/>` 
or for HTML5 `<input type="email" name="email" value=""/>`

Optional field should be `name`: `<input type="text" name="name" value=""/>`

Both fields will be merged to the `From` Email address: `"name" <email@address>`.

If you do not provide the name field, only the email address will be used.

### Testing


#### JavaScript

To do a quick test of the configuration for mailout on your Caddy server, the
following plain vanilla JavaScript using XMLHttpRequest does the job.

```
var xhr = new XMLHttpRequest();

xhr.open('POST', '/mailout'); // Change if /mailout is not your configured endpoint
xhr.onreadystatechange = function () { console.log(this.responseText); }

// Use this header for the paramater format below
xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
var params = 'name=Matt&email=' + encodeURIComponent('matt@github.com');

// Send the email
xhr.send(params);
```

#### CURL

If you are on a commandline console, the following CURL command also issues a
POST request with the name and email fields:

```
curl http://example.com/mailout -d 'name=Matt&email=matt@github.com'
```

### GMail

If you use Gmail as outgoing server these pages can help:

- [Google SMTP settings to send mail from a printer, scanner, or app](https://support.google.com/a/answer/176600)
- [Allowing less secure apps to access your account](https://support.google.com/accounts/answer/6010255)

I need some help to pimp the authentication feature of
[gomail](https://github.com/go-gomail/gomail/blob/master/smtp.go#L41) to avoid
switching on the less secure "feature".



# Todo

- file uploads
- implement ideas and improvements from open issues

CORS please use the dedicated Caddy module for handling CORS requests.

# Contribute

Send me a pull request or open an issue if you encounter a bug or something can
be improved!

Multi-time pull request senders gets collaborator access.

# License

[Cyrill Schumacher](https://github.com/SchumacherFM) - [My pgp public key](https://www.schumacher.fm/cyrill.asc)

Copyright 2016 Cyrill Schumacher All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use
this file except in compliance with the License. You may obtain a copy of the
License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed
under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied. See the License for the
specific language governing permissions and limitations under the License.
