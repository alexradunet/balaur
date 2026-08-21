routerAdd(
  "GET",
  "/api/balaur/household-invitations",
  function (event) {
    return require(__hooks + "/household_invitations.js").listInvitations(event)
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "POST",
  "/api/balaur/household-invitations",
  function (event) {
    return require(__hooks + "/household_invitations.js").createInvitation(event)
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "POST",
  "/api/balaur/household-invitations/{id}/cancel",
  function (event) {
    return require(__hooks + "/household_invitations.js").cancelInvitation(event)
  },
  $apis.requireAuth("members"),
)

routerAdd(
  "POST",
  "/api/balaur/household-invitations/redeem",
  function (event) {
    return require(__hooks + "/household_invitations.js").redeemInvitation(event)
  },
)
