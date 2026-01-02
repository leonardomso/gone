# Non-HTTP Links

These links should be skipped (not HTTP/HTTPS):

## Images with non-HTTP URLs

![Data URI](data:image/png;base64,ABC123)
![Relative Image](./images/logo.png)
![Root Relative](/images/icon.png)

## Autolinks with non-HTTP URLs

Email: <mailto:test@example.com>
FTP: <ftp://files.example.com>

## HTML links with non-HTTP URLs

<a href="/relative/path">Relative Path</a>
<a href="javascript:void(0)">JavaScript Link</a>
<a href="#section">Anchor Link</a>
<a href="mailto:contact@example.com">Email Link</a>

## Real HTTP links (should be extracted)

[Real Link](http://real.example.com)
![Real Image](https://cdn.example.com/image.png)
<a href="https://html.example.com">HTML Link</a>
