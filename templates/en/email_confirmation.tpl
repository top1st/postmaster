<html>
  <body>
    <p>Hello {{ .user.email }}!</p>
    <p>
      Use this unique link to confirm your email:
      <a href="{{ .domain }}/accounts/confirmation?confirmation_token={{ .token }}&lang=en">Confirm</a>
    </p>
  </body>
</html>
