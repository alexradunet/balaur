migrate(
  function (app) {
    var settings = app.findFirstRecordByData(
      "household_settings",
      "singletonKey",
      "household",
    )
    var connections = new Collection({
      type: "base",
      name: "calendar_connections",
      listRule: null,
      viewRule: null,
      createRule: null,
      updateRule: null,
      deleteRule: null,
      fields: [
        {
          type: "text",
          name: "singletonKey",
          required: true,
          max: 32,
          hidden: true,
        },
        {
          type: "select",
          name: "provider",
          required: true,
          maxSelect: 1,
          values: ["google"],
        },
        {
          type: "select",
          name: "status",
          required: true,
          maxSelect: 1,
          values: [
            "disconnected",
            "authorizing",
            "awaiting_selection",
            "connected",
            "authorization_failed",
          ],
        },
        {
          type: "text",
          name: "timeZone",
          required: true,
          min: 1,
          max: 100,
        },
        {
          type: "text",
          name: "selectedCalendarId",
          max: 1024,
        },
        {
          type: "text",
          name: "selectedCalendarName",
          max: 1024,
        },
        {
          type: "text",
          name: "selectedCalendarColor",
          max: 32,
        },
        {
          type: "text",
          name: "refreshTokenCiphertext",
          max: 16384,
          hidden: true,
        },
        {
          type: "text",
          name: "oauthStateHash",
          max: 64,
          hidden: true,
        },
        {
          type: "date",
          name: "oauthStateExpiresAt",
          hidden: true,
        },
        {
          type: "text",
          name: "pendingCalendarsJson",
          max: 200000,
          hidden: true,
        },
      ],
      indexes: [
        "CREATE UNIQUE INDEX idx_calendar_connections_singleton ON calendar_connections (singletonKey)",
      ],
    })
    app.save(connections)

    var connection = new Record(connections)
    connection.set("singletonKey", "household")
    connection.set("provider", "google")
    connection.set("status", "disconnected")
    connection.set("timeZone", settings.getString("timeZone"))
    app.save(connection)
  },
  function (app) {
    app.delete(app.findCollectionByNameOrId("calendar_connections"))
  },
)
