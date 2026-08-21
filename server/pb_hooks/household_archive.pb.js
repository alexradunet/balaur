routerAdd(
  "POST",
  "/api/balaur/household-archive",
  function (event) {
    return require(__hooks + "/household_archive.js").exportArchive(event)
  },
  $apis.requireAuth("members"),
)
