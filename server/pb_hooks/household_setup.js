var householdAdministratorRole = "household_administrator"
var householdSettingsKey = "household"
var setupRoute = "/api/balaur/setup"
var setupSecretPattern = /^[A-Za-z0-9]{32,128}$/

function noStore(event) {
  var headers = event.response.header()
  headers.set("Cache-Control", "no-store")
  headers.set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
  headers.set("Referrer-Policy", "no-referrer")
  headers.set("X-Content-Type-Options", "nosniff")
}

function householdSettings(app) {
  return app.findFirstRecordByData(
    "household_settings",
    "singletonKey",
    householdSettingsKey,
  )
}

function hasHouseholdAdministrator(app) {
  return app.countRecords(
    "members",
    $dbx.hashExp({ role: householdAdministratorRole }),
  ) > 0
}

function setupIsComplete(app) {
  return householdSettings(app).getString("setupCompletedAt") !== ""
}

function setupIsAvailable(app) {
  return !setupIsComplete(app) && !hasHouseholdAdministrator(app)
}

function setupExpiresAt() {
  var rawValue = $os.getenv("BALAUR_SETUP_EXPIRES_AT")
  if (!/^[0-9]+$/.test(rawValue)) {
    return 0
  }
  return parseInt(rawValue, 10)
}

function setupCapabilityIsActive() {
  var secret = $os.getenv("BALAUR_SETUP_SECRET")
  return setupSecretPattern.test(secret) && setupExpiresAt() * 1000 > Date.now()
}

function requireAvailableSetup(app) {
  if (!setupIsAvailable(app)) {
    throw new ApiError(409, "Household setup is complete.")
  }
}

function requireValidSetupCapability(token) {
  var expectedSecret = $os.getenv("BALAUR_SETUP_SECRET")
  if (!setupSecretPattern.test(expectedSecret)) {
    throw new NotFoundError("The setup link is invalid.")
  }

  if (setupExpiresAt() * 1000 <= Date.now()) {
    throw new ApiError(410, "The setup link expired.")
  }

  if (!setupSecretPattern.test(token)) {
    throw new NotFoundError("The setup link is invalid.")
  }

  var expectedHash = $security.sha256(expectedSecret)
  var suppliedHash = $security.sha256(token)
  if (!$security.equal(expectedHash, suppliedHash)) {
    throw new NotFoundError("The setup link is invalid.")
  }
}

function publicSetupUrl() {
  var publicUrl = $os.getenv("BALAUR_PUBLIC_URL").trim()
  if (publicUrl === "") {
    publicUrl = "http://127.0.0.1:8090"
  }
  return publicUrl.replace(/\/+$/, "") + setupRoute + "?token=" + $os.getenv("BALAUR_SETUP_SECRET")
}

function htmlEscape(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/\"/g, "&quot;")
    .replace(/'/g, "&#39;")
}

function setupPage(token) {
  return "<!doctype html>" +
    "<html lang=\"en\"><head><meta charset=\"utf-8\">" +
    "<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">" +
    "<meta name=\"referrer\" content=\"no-referrer\">" +
    "<title>Set up the Household</title>" +
    "<style>body{font:16px system-ui,sans-serif;max-width:34rem;margin:3rem auto;padding:0 1rem;color:#202124}" +
    "form{display:grid;gap:1rem}label{display:grid;gap:.35rem;font-weight:600}" +
    "input{font:inherit;padding:.7rem;border:1px solid #777;border-radius:.35rem}" +
    "button{font:inherit;padding:.8rem;border:0;border-radius:.35rem;background:#1f5fbf;color:white;font-weight:700;cursor:pointer}" +
    "small{font-weight:400}</style></head><body>" +
    "<main><h1>Create the first Household Administrator</h1>" +
    "<p>This link expires soon. You can use it one time.</p>" +
    "<form method=\"post\" action=\"" + setupRoute + "\">" +
    "<input type=\"hidden\" name=\"token\" value=\"" + htmlEscape(token) + "\">" +
    "<label>Household name<input name=\"householdName\" maxlength=\"120\" required autocomplete=\"organization\"></label>" +
    "<label>Household time zone<small>Use an Internet Assigned Numbers Authority time zone, such as Europe/Bucharest.</small>" +
    "<input name=\"timeZone\" maxlength=\"100\" required value=\"UTC\" autocomplete=\"off\"></label>" +
    "<label>Your name<input name=\"displayName\" maxlength=\"120\" required autocomplete=\"name\"></label>" +
    "<label>Email address<input type=\"email\" name=\"email\" maxlength=\"254\" required autocomplete=\"email\"></label>" +
    "<label>Password<small>Use at least 8 characters.</small>" +
    "<input type=\"password\" name=\"password\" minlength=\"8\" maxlength=\"71\" required autocomplete=\"new-password\"></label>" +
    "<label>Confirm password<input type=\"password\" name=\"passwordConfirm\" minlength=\"8\" maxlength=\"71\" required autocomplete=\"new-password\"></label>" +
    "<button type=\"submit\">Create the Household Administrator</button>" +
    "</form></main></body></html>"
}

function setupSuccessPage() {
  return "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\">" +
    "<meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">" +
    "<title>Household setup is complete</title></head><body>" +
    "<main><h1>Household setup is complete</h1>" +
    "<p>The Household Administrator exists. You can close this page.</p>" +
    "</main></body></html>"
}

function setupInput(event) {
  var body = event.requestInfo().body
  return {
    token: String(body.token || ""),
    householdName: String(body.householdName || "").trim(),
    timeZone: String(body.timeZone || "").trim(),
    displayName: String(body.displayName || "").trim(),
    email: String(body.email || "").trim().toLowerCase(),
    password: String(body.password || ""),
    passwordConfirm: String(body.passwordConfirm || ""),
  }
}

function validateSetupInput(input) {
  if (input.householdName.length < 1 || input.householdName.length > 120) {
    throw new BadRequestError("Enter a valid Household name.")
  }
  if (!/^(UTC|[A-Za-z0-9_+\-]+(?:\/[A-Za-z0-9_+\-]+)+)$/.test(input.timeZone)) {
    throw new BadRequestError("Enter a valid Household time zone.")
  }
  if (input.displayName.length < 1 || input.displayName.length > 120) {
    throw new BadRequestError("Enter a valid name.")
  }
  if (input.email.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(input.email)) {
    throw new BadRequestError("Enter a valid email address.")
  }
  if (input.password.length < 8 || input.password.length > 71) {
    throw new BadRequestError("Use a password with 8 to 71 characters.")
  }
  if (input.password !== input.passwordConfirm) {
    throw new BadRequestError("The passwords do not match.")
  }
}

function logSetupLink(app) {
  if (setupIsAvailable(app) && setupCapabilityIsActive()) {
    console.log("Household setup link: " + publicSetupUrl())
  }
}

function showSetup(event) {
  noStore(event)
  requireAvailableSetup(event.app)
  var token = String(event.requestInfo().query.token || "")
  requireValidSetupCapability(token)
  return event.html(200, setupPage(token))
}

function submitSetup(event) {
  noStore(event)
  requireAvailableSetup(event.app)

  var input = setupInput(event)
  requireValidSetupCapability(input.token)
  validateSetupInput(input)

  event.app.runInTransaction(function (transactionApp) {
    requireAvailableSetup(transactionApp)

    var members = transactionApp.findCollectionByNameOrId("members")
    var administrator = new Record(members)
    administrator.setEmail(input.email)
    administrator.setPassword(input.password)
    administrator.setVerified(true)
    administrator.set("displayName", input.displayName)
    administrator.set("role", householdAdministratorRole)
    transactionApp.save(administrator)

    var settings = householdSettings(transactionApp)
    settings.set("householdName", input.householdName)
    settings.set("timeZone", input.timeZone)
    settings.set("setupCompletedAt", new Date().toISOString())
    transactionApp.save(settings)
  })

  return event.html(201, setupSuccessPage())
}

module.exports = {
  logSetupLink: logSetupLink,
  showSetup: showSetup,
  submitSetup: submitSetup,
}
