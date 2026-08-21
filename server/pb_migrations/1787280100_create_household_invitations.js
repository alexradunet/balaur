migrate(
  function (app) {
    var members = app.findCollectionByNameOrId("members")
    var invitations = new Collection({
      type: "base",
      name: "household_invitations",
      listRule: null,
      viewRule: null,
      createRule: null,
      updateRule: null,
      deleteRule: null,
      fields: [
        {
          type: "text",
          name: "tokenHash",
          required: true,
          min: 64,
          max: 64,
          pattern: "^[a-f0-9]{64}$",
          hidden: true,
        },
        {
          type: "relation",
          name: "creator",
          required: true,
          maxSelect: 1,
          collectionId: members.id,
          cascadeDelete: false,
        },
        {
          type: "date",
          name: "expiresAt",
          required: true,
        },
        {
          type: "select",
          name: "role",
          required: true,
          maxSelect: 1,
          values: ["member", "household_administrator"],
        },
        {
          type: "select",
          name: "state",
          required: true,
          maxSelect: 1,
          values: ["active", "used", "canceled"],
        },
      ],
      indexes: [
        "CREATE UNIQUE INDEX idx_household_invitations_token_hash ON household_invitations (tokenHash)",
        "CREATE INDEX idx_household_invitations_state_expires ON household_invitations (state, expiresAt)",
      ],
    })
    app.save(invitations)
  },
  function (app) {
    app.delete(app.findCollectionByNameOrId("household_invitations"))
  },
)
