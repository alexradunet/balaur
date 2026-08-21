var householdAdministratorRole = "household_administrator"
var connectionCollection = "calendar_connections"
var stateLifetimeMilliseconds = 10 * 60 * 1000
var statePattern = /^[A-Za-z0-9]{64}$/

function noStore(event) {
  event.response.header().set("Cache-Control", "no-store")
  event.response.header().set("X-Content-Type-Options", "nosniff")
  event.response.header().set("Referrer-Policy", "no-referrer")
}

function requireHouseholdAdministrator(event) {
  if (
    !event.auth ||
    event.auth.getString("role") !== householdAdministratorRole
  ) {
    throw new ForbiddenError(
      "Only a Household Administrator can manage the Calendar Connection.",
    )
  }
  return event.auth
}

function connectionRecord(app) {
  return app.findFirstRecordByData(
    connectionCollection,
    "singletonKey",
    "household",
  )
}

function currentTimeZone(app) {
  return app
    .findFirstRecordByData("household_settings", "singletonKey", "household")
    .getString("timeZone")
}

function pendingCalendars(connection) {
  var value = connection.getString("pendingCalendarsJson")
  if (value === "") {
    return []
  }
  try {
    var parsed = JSON.parse(value)
    return Array.isArray(parsed) ? parsed : []
  } catch (error) {
    return []
  }
}

function publicCalendar(calendar) {
  var result = {
    id: String(calendar.id),
    name: String(calendar.name),
  }
  if (calendar.colorHex) {
    result.colorHex = String(calendar.colorHex)
  }
  return result
}

function publicConnection(connection, includeAvailableCalendars) {
  var selectedCalendar = null
  if (connection.getString("selectedCalendarId") !== "") {
    selectedCalendar = {
      id: connection.getString("selectedCalendarId"),
      name: connection.getString("selectedCalendarName"),
    }
    var selectedColor = connection.getString("selectedCalendarColor")
    if (selectedColor !== "") {
      selectedCalendar.colorHex = selectedColor
    }
  }
  return {
    provider: connection.getString("provider"),
    status: connection.getString("status"),
    householdTimeZone: connection.getString("timeZone"),
    selectedCalendar: selectedCalendar,
    availableCalendars: includeAvailableCalendars
      ? pendingCalendars(connection).map(publicCalendar)
      : [],
  }
}

function clearAuthorization(connection) {
  connection.set("refreshTokenCiphertext", "")
  connection.set("oauthStateHash", "")
  connection.set("oauthStateExpiresAt", "")
  connection.set("pendingCalendarsJson", "")
}

function clearSelection(connection) {
  connection.set("selectedCalendarId", "")
  connection.set("selectedCalendarName", "")
  connection.set("selectedCalendarColor", "")
}

function encryptionKey() {
  var key = $os.getenv("BALAUR_CALENDAR_ENCRYPTION_KEY")
  if (key.length !== 32) {
    throw new ApiError(
      503,
      "The Calendar Connection encryption key is not configured.",
    )
  }
  return key
}

function loadConnection(event) {
  noStore(event)
  var connection = connectionRecord(event.app)
  var canSelect =
    event.auth &&
    event.auth.getString("role") === householdAdministratorRole &&
    connection.getString("status") === "awaiting_selection"
  return event.json(200, {
    connection: publicConnection(connection, canSelect),
  })
}

function beginAuthorization(event) {
  noStore(event)
  requireHouseholdAdministrator(event)
  encryptionKey()
  var source = require(__hooks + "/calendar_source_adapter.js")
  var adapter = source.createCalendarSourceAdapter()
  var connection = connectionRecord(event.app)
  var body = event.requestInfo().body
  var replace = body.replace === true
  if (connection.getString("status") === "connected" && !replace) {
    throw new ApiError(409, "Replace the current Calendar Connection first.")
  }

  var state = $security.randomString(64)
  connection.set("provider", "google")
  connection.set("status", "authorizing")
  connection.set("timeZone", currentTimeZone(event.app))
  clearAuthorization(connection)
  clearSelection(connection)
  connection.set("oauthStateHash", $security.sha256(state))
  connection.set(
    "oauthStateExpiresAt",
    new Date(Date.now() + stateLifetimeMilliseconds).toISOString(),
  )
  event.app.save(connection)

  return event.json(200, {
    authorizationUrl: adapter.authorizationUrl(state),
    requestedScopes: source.scopes,
    connection: publicConnection(connection, false),
  })
}

function validState(connection, state) {
  if (!statePattern.test(state)) {
    return false
  }
  var expectedHash = connection.getString("oauthStateHash")
  var expiresAt = new Date(
    connection.getString("oauthStateExpiresAt"),
  ).getTime()
  return (
    connection.getString("status") === "authorizing" &&
    expectedHash !== "" &&
    expiresAt > Date.now() &&
    $security.equal($security.sha256(state), expectedHash)
  )
}

function failAuthorization(app, connection) {
  connection.set("status", "authorization_failed")
  clearAuthorization(connection)
  clearSelection(connection)
  app.save(connection)
}

function authorizationResultPage(event, success, message) {
  var title = success
    ? "Google Calendar authorization is complete"
    : "Google Calendar authorization failed"
  return event.html(
    200,
    "<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width\"><title>" +
      title +
      "</title></head><body><main><h1>" +
      title +
      "</h1><p>" +
      message +
      "</p><p>You can return to Balaur.</p></main></body></html>",
  )
}

function authorizationCallback(event) {
  noStore(event)
  var connection = connectionRecord(event.app)
  var state = event.request.url.query().get("state")
  if (!validState(connection, state)) {
    throw new BadRequestError("The Calendar Connection state is invalid.")
  }

  var denial = event.request.url.query().get("error")
  if (denial !== "") {
    failAuthorization(event.app, connection)
    return authorizationResultPage(
      event,
      false,
      "Google did not authorize the Calendar Connection.",
    )
  }

  var code = event.request.url.query().get("code")
  if (code === "") {
    failAuthorization(event.app, connection)
    return authorizationResultPage(
      event,
      false,
      "Google did not return an authorization code.",
    )
  }

  try {
    var source = require(__hooks + "/calendar_source_adapter.js")
    var adapter = source.createCalendarSourceAdapter()
    var tokens = adapter.exchangeAuthorization(code)
    var calendars = adapter.listCalendars(tokens.accessToken)
    if (calendars.length === 0) {
      throw new Error("The authorized account has no readable calendars.")
    }
    connection.set(
      "refreshTokenCiphertext",
      $security.encrypt(tokens.refreshToken, encryptionKey()),
    )
    connection.set(
      "pendingCalendarsJson",
      JSON.stringify(calendars.map(publicCalendar)),
    )
    connection.set("status", "awaiting_selection")
    connection.set("oauthStateHash", "")
    connection.set("oauthStateExpiresAt", "")
    event.app.save(connection)
    return authorizationResultPage(
      event,
      true,
      "Select one shared calendar in Balaur.",
    )
  } catch (error) {
    failAuthorization(event.app, connection)
    return authorizationResultPage(
      event,
      false,
      "Balaur could not complete Google authorization.",
    )
  }
}

function selectCalendar(event) {
  noStore(event)
  requireHouseholdAdministrator(event)
  var connection = connectionRecord(event.app)
  if (connection.getString("status") !== "awaiting_selection") {
    throw new ApiError(409, "Authorize Google before you select a calendar.")
  }
  var calendarId = String(event.requestInfo().body.calendarId || "")
  var selected = null
  pendingCalendars(connection).forEach(function (calendar) {
    if (calendar.id === calendarId) {
      selected = publicCalendar(calendar)
    }
  })
  if (!selected) {
    throw new BadRequestError("Select a calendar from the authorized list.")
  }

  connection.set("selectedCalendarId", selected.id)
  connection.set("selectedCalendarName", selected.name)
  connection.set("selectedCalendarColor", selected.colorHex || "")
  connection.set("pendingCalendarsJson", "")
  connection.set("status", "connected")
  event.app.save(connection)
  return event.json(200, {
    connection: publicConnection(connection, false),
  })
}

function disconnect(event) {
  noStore(event)
  requireHouseholdAdministrator(event)
  var connection = connectionRecord(event.app)
  clearAuthorization(connection)
  clearSelection(connection)
  connection.set("status", "disconnected")
  connection.set("timeZone", currentTimeZone(event.app))
  event.app.save(connection)
  return event.json(200, {
    connection: publicConnection(connection, false),
  })
}

function entryRange(event) {
  var timeMin = event.request.url.query().get("timeMin")
  var timeMax = event.request.url.query().get("timeMax")
  var start = new Date(timeMin)
  var end = new Date(timeMax)
  var length = end.getTime() - start.getTime()
  if (
    !timeMin ||
    !timeMax ||
    isNaN(start.getTime()) ||
    isNaN(end.getTime()) ||
    length <= 0 ||
    length > 62 * 24 * 60 * 60 * 1000
  ) {
    throw new BadRequestError(
      "Request a positive Calendar Entry range of at most 62 days.",
    )
  }
  return {
    timeMin: start.toISOString(),
    timeMax: end.toISOString(),
  }
}

function loadEntries(event) {
  noStore(event)
  var range = entryRange(event)
  var connection = connectionRecord(event.app)
  if (
    connection.getString("status") !== "connected" ||
    connection.getString("refreshTokenCiphertext") === "" ||
    connection.getString("selectedCalendarId") === ""
  ) {
    throw new ApiError(424, "The Calendar Connection is disconnected.")
  }
  var source = require(__hooks + "/calendar_source_adapter.js")
  var adapter = source.createCalendarSourceAdapter()
  try {
    var refreshToken = $security.decrypt(
      connection.getString("refreshTokenCiphertext"),
      encryptionKey(),
    )
    var accessToken = adapter.refreshAccessToken(refreshToken)
    var entries = adapter.listEntries({
      accessToken: accessToken,
      calendar: {
        id: connection.getString("selectedCalendarId"),
        name: connection.getString("selectedCalendarName"),
        colorHex: connection.getString("selectedCalendarColor"),
      },
      timeMin: range.timeMin,
      timeMax: range.timeMax,
      timeZone: connection.getString("timeZone"),
    })
    return event.json(200, {items: entries})
  } catch (error) {
    if (error && error.authorizationFailure === true) {
      clearAuthorization(connection)
      clearSelection(connection)
      connection.set("status", "disconnected")
      event.app.save(connection)
      throw new ApiError(424, "The Calendar Connection authorization failed.")
    }
    throw new ApiError(502, "The Calendar Source request failed.")
  }
}

function revokeFake(event) {
  noStore(event)
  requireHouseholdAdministrator(event)
  var source = require(__hooks + "/calendar_source_adapter.js")
  if (!source.fakeAdapterIsEnabled()) {
    throw new NotFoundError("The fake Calendar Source is not enabled.")
  }
  var connection = connectionRecord(event.app)
  connection.set(
    "refreshTokenCiphertext",
    $security.encrypt("revoked-fake-token", encryptionKey()),
  )
  event.app.save(connection)
  return event.noContent(204)
}

function authorizeFake(event) {
  noStore(event)
  var source = require(__hooks + "/calendar_source_adapter.js")
  if (!source.fakeAdapterIsEnabled()) {
    throw new NotFoundError("The fake Calendar Source is not enabled.")
  }
  var state = event.request.url.query().get("state")
  var decision = event.request.url.query().get("decision")
  var target = source.callbackUrl() + "?state=" + encodeURIComponent(state)
  if (decision === "deny") {
    target += "&error=access_denied"
  } else {
    target += "&code=fake-authorization-code"
  }
  return event.redirect(302, target)
}

module.exports = {
  authorizationCallback: authorizationCallback,
  authorizeFake: authorizeFake,
  beginAuthorization: beginAuthorization,
  disconnect: disconnect,
  loadConnection: loadConnection,
  loadEntries: loadEntries,
  revokeFake: revokeFake,
  selectCalendar: selectCalendar,
}
