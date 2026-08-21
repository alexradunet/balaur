migrate(
  function (app) {
    var members = new Collection({
      type: "auth",
      name: "members",
      listRule: null,
      viewRule: "id = @request.auth.id",
      createRule: null,
      updateRule: null,
      deleteRule: null,
      authRule: "",
      manageRule: null,
      fields: [
        {
          type: "text",
          name: "displayName",
          required: true,
          min: 1,
          max: 120,
          presentable: true,
        },
        {
          type: "select",
          name: "role",
          required: true,
          maxSelect: 1,
          values: ["member", "household_administrator"],
        },
      ],
      passwordAuth: {
        enabled: true,
        identityFields: ["email"],
      },
    })
    app.save(members)

    var settings = new Collection({
      type: "base",
      name: "household_settings",
      listRule: "@request.auth.id != ''",
      viewRule: "@request.auth.id != ''",
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
          type: "text",
          name: "householdName",
          required: true,
          min: 1,
          max: 120,
        },
        {
          type: "text",
          name: "timeZone",
          required: true,
          min: 1,
          max: 100,
        },
        {
          type: "date",
          name: "setupCompletedAt",
          hidden: true,
        },
      ],
      indexes: [
        "CREATE UNIQUE INDEX idx_household_settings_singleton ON household_settings (singletonKey)",
      ],
    })
    app.save(settings)

    var householdSettings = new Record(settings)
    householdSettings.set("singletonKey", "household")
    householdSettings.set("householdName", "Household")
    householdSettings.set("timeZone", "UTC")
    app.save(householdSettings)
  },
  function (app) {
    app.delete(app.findCollectionByNameOrId("household_settings"))
    app.delete(app.findCollectionByNameOrId("members"))
  },
)
