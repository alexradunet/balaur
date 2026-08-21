routerAdd(
  "GET",
  "/api/balaur/calendar-connection",
  function (event) {
    return require(__hooks + "/calendar_connection.js").loadConnection(event)
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "POST",
  "/api/balaur/calendar-connection/authorize",
  function (event) {
    return require(__hooks + "/calendar_connection.js").beginAuthorization(
      event,
    )
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "POST",
  "/api/balaur/calendar-connection/select",
  function (event) {
    return require(__hooks + "/calendar_connection.js").selectCalendar(event)
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "POST",
  "/api/balaur/calendar-connection/disconnect",
  function (event) {
    return require(__hooks + "/calendar_connection.js").disconnect(event)
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "GET",
  "/api/balaur/calendar-entries",
  function (event) {
    return require(__hooks + "/calendar_connection.js").loadEntries(event)
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "POST",
  "/api/balaur/testing/calendar-source/revoke",
  function (event) {
    return require(__hooks + "/calendar_connection.js").revokeFake(event)
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "GET",
  "/api/balaur/calendar-source/callback",
  function (event) {
    return require(__hooks + "/calendar_connection.js").authorizationCallback(
      event,
    )
  },
)

routerAdd(
  "GET",
  "/api/balaur/testing/calendar-source/authorize",
  function (event) {
    return require(__hooks + "/calendar_connection.js").authorizeFake(event)
  },
)
