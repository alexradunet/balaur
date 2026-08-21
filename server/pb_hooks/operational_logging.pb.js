routerUse(function (event) {
  var startedAt = Date.now()
  try {
    return event.next()
  } finally {
    console.log(
      JSON.stringify({
        event: "household_request",
        method: event.request.method,
        path: event.request.url.path,
        durationMilliseconds: Date.now() - startedAt,
      }),
    )
  }
})
