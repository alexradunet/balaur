var googleCalendarListScope =
  "https://www.googleapis.com/auth/calendar.calendarlist.readonly"
var googleCalendarEventsScope =
  "https://www.googleapis.com/auth/calendar.events.readonly"

function normalizedPublicUrl() {
  var value = $os.getenv("BALAUR_PUBLIC_URL").trim()
  if (value === "") {
    value = "http://127.0.0.1:8090"
  }
  return value.replace(/\/+$/, "")
}

function callbackUrl() {
  return normalizedPublicUrl() + "/api/balaur/calendar-source/callback"
}

function requireGoogleConfiguration() {
  var clientId = $os.getenv("BALAUR_GOOGLE_OAUTH_CLIENT_ID").trim()
  var clientSecret = $os.getenv("BALAUR_GOOGLE_OAUTH_CLIENT_SECRET").trim()
  if (clientId === "" || clientSecret === "") {
    throw new ApiError(503, "Google Calendar authorization is not configured.")
  }
  return {
    clientId: clientId,
    clientSecret: clientSecret,
  }
}

function formBody(values) {
  return Object.keys(values)
    .map(function (name) {
      return encodeURIComponent(name) + "=" + encodeURIComponent(values[name])
    })
    .join("&")
}

function authorizationError(message) {
  var error = new Error(message)
  error.authorizationFailure = true
  return error
}

function providerEntry(item, calendar) {
  if (!item || item.status === "cancelled" || !item.start || !item.end) {
    return null
  }
  var allDay = Boolean(item.start.date)
  var start = allDay ? item.start.date : item.start.dateTime
  var end = allDay ? item.end.date : item.end.dateTime
  if (!start || !end) {
    return null
  }
  var title = String(item.summary || "").trim()
  return {
    id: String(item.id || item.recurringEventId || start),
    title: title === "" ? "Busy" : title,
    start: String(start),
    end: String(end),
    allDay: allDay,
    location: item.location ? String(item.location) : "",
    colorHex: calendar.colorHex || "#4285f4",
  }
}

function collectEntries(fetchPage, calendar) {
  var entries = []
  var pageToken = ""
  do {
    var page = fetchPage(pageToken)
    ;(page.items || []).forEach(function (item) {
      var entry = providerEntry(item, calendar)
      if (entry) {
        entries.push(entry)
      }
    })
    pageToken = page.nextPageToken ? String(page.nextPageToken) : ""
  } while (pageToken !== "")
  entries.sort(function (left, right) {
    return new Date(left.start).getTime() - new Date(right.start).getTime()
  })
  return entries
}

function googleAdapter() {
  return {
    authorizationUrl: function (state) {
      var config = requireGoogleConfiguration()
      return (
        "https://accounts.google.com/o/oauth2/v2/auth?" +
        formBody({
          access_type: "offline",
          client_id: config.clientId,
          include_granted_scopes: "false",
          prompt: "consent",
          redirect_uri: callbackUrl(),
          response_type: "code",
          scope: googleCalendarListScope + " " + googleCalendarEventsScope,
          state: state,
        })
      )
    },
    exchangeAuthorization: function (code) {
      var config = requireGoogleConfiguration()
      var response = $http.send({
        method: "POST",
        url: "https://oauth2.googleapis.com/token",
        headers: {"Content-Type": "application/x-www-form-urlencoded"},
        body: formBody({
          client_id: config.clientId,
          client_secret: config.clientSecret,
          code: code,
          grant_type: "authorization_code",
          redirect_uri: callbackUrl(),
        }),
        timeout: 30,
      })
      if (
        response.statusCode !== 200 ||
        !response.json ||
        !response.json.access_token ||
        !response.json.refresh_token
      ) {
        throw new Error("Google rejected the authorization code.")
      }
      return {
        accessToken: String(response.json.access_token),
        refreshToken: String(response.json.refresh_token),
      }
    },
    refreshAccessToken: function (refreshToken) {
      var config = requireGoogleConfiguration()
      var response = $http.send({
        method: "POST",
        url: "https://oauth2.googleapis.com/token",
        headers: {"Content-Type": "application/x-www-form-urlencoded"},
        body: formBody({
          client_id: config.clientId,
          client_secret: config.clientSecret,
          grant_type: "refresh_token",
          refresh_token: refreshToken,
        }),
        timeout: 30,
      })
      if (
        response.statusCode !== 200 ||
        !response.json ||
        !response.json.access_token
      ) {
        throw authorizationError("Google rejected the refresh token.")
      }
      return String(response.json.access_token)
    },
    listCalendars: function (accessToken) {
      var calendars = []
      var pageToken = ""
      do {
        var url =
          "https://www.googleapis.com/calendar/v3/users/me/calendarList?" +
          "maxResults=250&minAccessRole=reader"
        if (pageToken !== "") {
          url += "&pageToken=" + encodeURIComponent(pageToken)
        }
        var response = $http.send({
          method: "GET",
          url: url,
          headers: {Authorization: "Bearer " + accessToken},
          timeout: 30,
        })
        if (response.statusCode !== 200 || !response.json) {
          throw new Error("Google Calendar did not return the calendar list.")
        }
        var items = response.json.items || []
        items.forEach(function (item) {
          if (item.id && item.summary) {
            calendars.push({
              id: String(item.id),
              name: String(item.summary),
              colorHex: item.backgroundColor
                ? String(item.backgroundColor)
                : "",
            })
          }
        })
        pageToken = response.json.nextPageToken
          ? String(response.json.nextPageToken)
          : ""
      } while (pageToken !== "")
      return calendars
    },
    listEntries: function (options) {
      return collectEntries(
        function (pageToken) {
          var url =
            "https://www.googleapis.com/calendar/v3/calendars/" +
            encodeURIComponent(options.calendar.id) +
            "/events?" +
            formBody({
              maxResults: "2500",
              orderBy: "startTime",
              showDeleted: "false",
              singleEvents: "true",
              timeMax: options.timeMax,
              timeMin: options.timeMin,
              timeZone: options.timeZone,
            })
          if (pageToken !== "") {
            url += "&pageToken=" + encodeURIComponent(pageToken)
          }
          var response = $http.send({
            method: "GET",
            url: url,
            headers: {Authorization: "Bearer " + options.accessToken},
            timeout: 30,
          })
          if (response.statusCode === 401 || response.statusCode === 403) {
            throw authorizationError(
              "Google rejected the Calendar Connection.",
            )
          }
          if (response.statusCode !== 200 || !response.json) {
            throw new Error("Google Calendar did not return Calendar Entries.")
          }
          return response.json
        },
        options.calendar,
      )
    },
  }
}

function fakeAdapter() {
  return {
    authorizationUrl: function (state) {
      return (
        normalizedPublicUrl() +
        "/api/balaur/testing/calendar-source/authorize?state=" +
        encodeURIComponent(state)
      )
    },
    exchangeAuthorization: function (code) {
      if (code !== "fake-authorization-code") {
        throw new Error("The fake Calendar Source rejected the code.")
      }
      return {
        accessToken: "fake-access-token",
        refreshToken: "fake-refresh-token",
      }
    },
    refreshAccessToken: function (refreshToken) {
      if (refreshToken !== "fake-refresh-token") {
        throw authorizationError("The fake refresh token is invalid.")
      }
      return "fake-access-token"
    },
    listCalendars: function (accessToken) {
      if (accessToken !== "fake-access-token") {
        throw new Error("The fake Calendar Source rejected the token.")
      }
      return [
        {id: "family@example.com", name: "Family", colorHex: "#4285f4"},
        {id: "school@example.com", name: "School", colorHex: "#d50000"},
      ]
    },
    listEntries: function (options) {
      if (options.accessToken !== "fake-access-token") {
        throw authorizationError("The fake access token is invalid.")
      }
      var day = options.timeMin.slice(0, 10)
      var nextDay = new Date(
        new Date(day + "T00:00:00Z").getTime() + 24 * 60 * 60 * 1000,
      )
        .toISOString()
        .slice(0, 10)
      return collectEntries(
        function (pageToken) {
          if (pageToken === "") {
            return {
              items: [
                {
                  id: "timed-entry",
                  summary: "Family dinner",
                  start: {dateTime: day + "T18:00:00Z"},
                  end: {dateTime: day + "T19:30:00Z"},
                  location: "Home",
                },
                {
                  id: "cancelled-entry",
                  status: "cancelled",
                  summary: "Canceled",
                  start: {dateTime: day + "T12:00:00Z"},
                  end: {dateTime: day + "T13:00:00Z"},
                },
              ],
              nextPageToken: "second-page",
            }
          }
          return {
            items: [
              {
                id: "all-day-entry",
                recurringEventId: "weekly-family-day",
                summary: "",
                start: {date: day},
                end: {date: nextDay},
              },
            ],
          }
        },
        options.calendar,
      )
    },
  }
}

function createCalendarSourceAdapter() {
  if ($os.getenv("BALAUR_CALENDAR_SOURCE_ADAPTER") === "fake") {
    return fakeAdapter()
  }
  return googleAdapter()
}

function fakeAdapterIsEnabled() {
  return $os.getenv("BALAUR_CALENDAR_SOURCE_ADAPTER") === "fake"
}

module.exports = {
  callbackUrl: callbackUrl,
  createCalendarSourceAdapter: createCalendarSourceAdapter,
  fakeAdapterIsEnabled: fakeAdapterIsEnabled,
  scopes: [googleCalendarListScope, googleCalendarEventsScope],
}
