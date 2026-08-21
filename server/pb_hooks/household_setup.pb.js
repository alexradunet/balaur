$app.onServe().bindFunc(function (event) {
  if ($os.getenv("BALAUR_DISABLE_POCKETBASE_INSTALLER") === "1") {
    event.installerFunc = null
  }
  require(__hooks + "/household_setup.js").logSetupLink(event.app)
  event.next()
})

routerAdd("GET", "/api/balaur/setup", function (event) {
  return require(__hooks + "/household_setup.js").showSetup(event)
})

routerAdd("POST", "/api/balaur/setup", function (event) {
  return require(__hooks + "/household_setup.js").submitSetup(event)
})
