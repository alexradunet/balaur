var householdAdministratorRole = "household_administrator"

function noStore(event) {
  event.response.header().set("Cache-Control", "no-store")
  event.response.header().set("X-Content-Type-Options", "nosniff")
  event.response.header().set("Referrer-Policy", "no-referrer")
}

function requireConfirmedAdministrator(event) {
  if (
    !event.auth ||
    event.auth.getString("role") !== householdAdministratorRole
  ) {
    throw new ForbiddenError(
      "Only a Household Administrator can export a Household Archive.",
    )
  }
  var password = String(event.requestInfo().body.password || "")
  if (password === "" || !event.auth.validatePassword(password)) {
    throw new ApiError(401, "Confirm the Household Administrator password.")
  }
}

function utf8(value) {
  var bytes = []
  for (var index = 0; index < value.length; index += 1) {
    var code = value.charCodeAt(index)
    if (code >= 0xd800 && code <= 0xdbff && index + 1 < value.length) {
      var low = value.charCodeAt(index + 1)
      if (low >= 0xdc00 && low <= 0xdfff) {
        code = 0x10000 + ((code - 0xd800) << 10) + (low - 0xdc00)
        index += 1
      }
    }
    if (code < 0x80) {
      bytes.push(code)
    } else if (code < 0x800) {
      bytes.push(0xc0 | (code >> 6), 0x80 | (code & 0x3f))
    } else if (code < 0x10000) {
      bytes.push(
        0xe0 | (code >> 12),
        0x80 | ((code >> 6) & 0x3f),
        0x80 | (code & 0x3f),
      )
    } else {
      bytes.push(
        0xf0 | (code >> 18),
        0x80 | ((code >> 12) & 0x3f),
        0x80 | ((code >> 6) & 0x3f),
        0x80 | (code & 0x3f),
      )
    }
  }
  return bytes
}

function append16(output, value) {
  output.push(value & 0xff, (value >>> 8) & 0xff)
}

function append32(output, value) {
  output.push(
    value & 0xff,
    (value >>> 8) & 0xff,
    (value >>> 16) & 0xff,
    (value >>> 24) & 0xff,
  )
}

function crc32(bytes) {
  var crc = 0xffffffff
  for (var index = 0; index < bytes.length; index += 1) {
    crc ^= bytes[index]
    for (var bit = 0; bit < 8; bit += 1) {
      crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0)
    }
  }
  return (crc ^ 0xffffffff) >>> 0
}

function dosDateTime(date) {
  var year = Math.max(1980, date.getUTCFullYear())
  return {
    date:
      ((year - 1980) << 9) |
      ((date.getUTCMonth() + 1) << 5) |
      date.getUTCDate(),
    time:
      (date.getUTCHours() << 11) |
      (date.getUTCMinutes() << 5) |
      Math.floor(date.getUTCSeconds() / 2),
  }
}

function zip(files) {
  var output = []
  var central = []
  var now = dosDateTime(new Date())
  files.forEach(function (file) {
    var name = utf8(file.name)
    var data = file.bytes
    var checksum = crc32(data)
    var offset = output.length
    append32(output, 0x04034b50)
    append16(output, 20)
    append16(output, 0x0800)
    append16(output, 0)
    append16(output, now.time)
    append16(output, now.date)
    append32(output, checksum)
    append32(output, data.length)
    append32(output, data.length)
    append16(output, name.length)
    append16(output, 0)
    Array.prototype.push.apply(output, name)
    Array.prototype.push.apply(output, data)

    append32(central, 0x02014b50)
    append16(central, 20)
    append16(central, 20)
    append16(central, 0x0800)
    append16(central, 0)
    append16(central, now.time)
    append16(central, now.date)
    append32(central, checksum)
    append32(central, data.length)
    append32(central, data.length)
    append16(central, name.length)
    append16(central, 0)
    append16(central, 0)
    append16(central, 0)
    append16(central, 0)
    append32(central, file.name.slice(-1) === "/" ? 0x10 : 0)
    append32(central, offset)
    Array.prototype.push.apply(central, name)
  })
  var centralOffset = output.length
  Array.prototype.push.apply(output, central)
  append32(output, 0x06054b50)
  append16(output, 0)
  append16(output, 0)
  append16(output, files.length)
  append16(output, files.length)
  append32(output, central.length)
  append32(output, centralOffset)
  append16(output, 0)
  return output
}

function publicMembers(app) {
  return app.findAllRecords("members").map(function (member) {
    return {
      id: member.id,
      displayName: member.getString("displayName"),
      email: member.email(),
      role: member.getString("role"),
      created: member.getString("created"),
      updated: member.getString("updated"),
    }
  })
}

function publicInvitations(app) {
  return app.findAllRecords("household_invitations").map(function (invitation) {
    return {
      id: invitation.id,
      creatorId: invitation.getString("creator"),
      expiresAt: invitation.getString("expiresAt"),
      role: invitation.getString("role"),
      state: invitation.getString("state"),
    }
  })
}

function publicConnection(app) {
  var connection = app.findFirstRecordByData(
    "calendar_connections",
    "singletonKey",
    "household",
  )
  var selected = null
  if (connection.getString("selectedCalendarId") !== "") {
    selected = {
      id: connection.getString("selectedCalendarId"),
      name: connection.getString("selectedCalendarName"),
      colorHex: connection.getString("selectedCalendarColor"),
    }
  }
  return {
    provider: connection.getString("provider"),
    status: connection.getString("status"),
    householdTimeZone: connection.getString("timeZone"),
    selectedCalendar: selected,
  }
}

function householdState(app) {
  var settings = app.findFirstRecordByData(
    "household_settings",
    "singletonKey",
    "household",
  )
  return {
    household: {
      name: settings.getString("householdName"),
      timeZone: settings.getString("timeZone"),
    },
    invitations: publicInvitations(app),
    calendarConnection: publicConnection(app),
  }
}

function encryptionKey() {
  var key = $os.getenv("BALAUR_CALENDAR_ENCRYPTION_KEY")
  if (key.length !== 32) {
    throw new ApiError(503, "The Calendar encryption key is not configured.")
  }
  return key
}

function icalEscape(value) {
  return String(value || "")
    .replace(/\\/g, "\\\\")
    .replace(/\r?\n/g, "\\n")
    .replace(/,/g, "\\,")
    .replace(/;/g, "\\;")
}

function icalDate(value) {
  return value.replace(/-/g, "")
}

function icalInstant(value) {
  return new Date(value)
    .toISOString()
    .replace(/[-:]/g, "")
    .replace(/\.\d{3}Z$/, "Z")
}

function calendarSnapshot(app) {
  var connection = app.findFirstRecordByData(
    "calendar_connections",
    "singletonKey",
    "household",
  )
  if (
    connection.getString("status") !== "connected" ||
    connection.getString("refreshTokenCiphertext") === ""
  ) {
    throw new ApiError(409, "Connect a Calendar Source before export.")
  }
  var adapter = require(__hooks + "/calendar_source_adapter.js")
    .createCalendarSourceAdapter()
  var refreshToken = $security.decrypt(
    connection.getString("refreshTokenCiphertext"),
    encryptionKey(),
  )
  var accessToken = adapter.refreshAccessToken(refreshToken)
  var now = Date.now()
  var entries = adapter.listEntries({
    accessToken: accessToken,
    calendar: {
      id: connection.getString("selectedCalendarId"),
      name: connection.getString("selectedCalendarName"),
      colorHex: connection.getString("selectedCalendarColor"),
    },
    timeMin: new Date(now - 30 * 24 * 60 * 60 * 1000).toISOString(),
    timeMax: new Date(now + 366 * 24 * 60 * 60 * 1000).toISOString(),
    timeZone: connection.getString("timeZone"),
  })
  var lines = [
    "BEGIN:VCALENDAR",
    "VERSION:2.0",
    "PRODID:-//Balaur//Household Archive//EN",
    "CALSCALE:GREGORIAN",
  ]
  entries.forEach(function (entry) {
    lines.push("BEGIN:VEVENT", "UID:" + icalEscape(entry.id) + "@balaur")
    if (entry.allDay) {
      lines.push(
        "DTSTART;VALUE=DATE:" + icalDate(entry.start),
        "DTEND;VALUE=DATE:" + icalDate(entry.end),
      )
    } else {
      lines.push(
        "DTSTART:" + icalInstant(entry.start),
        "DTEND:" + icalInstant(entry.end),
      )
    }
    lines.push("SUMMARY:" + icalEscape(entry.title))
    if (entry.location) {
      lines.push("LOCATION:" + icalEscape(entry.location))
    }
    lines.push("END:VEVENT")
  })
  lines.push("END:VCALENDAR", "")
  return lines.join("\r\n")
}

function exportArchive(event) {
  noStore(event)
  requireConfirmedAdministrator(event)
  var body = event.requestInfo().body
  var includeCalendarSnapshot = body.includeCalendarSnapshot === true
  var createdAt = new Date().toISOString()
  var files = [
    {
      name: "manifest.json",
      bytes: utf8(
        JSON.stringify(
          {
            format: "balaur-household-archive",
            version: 1,
            createdAt: createdAt,
            contents: [
              "household-state.json",
              "members.json",
              "shared-files/",
            ].concat(includeCalendarSnapshot ? ["calendar.ics"] : []),
          },
          null,
          2,
        ) + "\n",
      ),
    },
    {
      name: "household-state.json",
      bytes: utf8(JSON.stringify(householdState(event.app), null, 2) + "\n"),
    },
    {
      name: "members.json",
      bytes: utf8(JSON.stringify(publicMembers(event.app), null, 2) + "\n"),
    },
    {name: "shared-files/", bytes: []},
  ]
  if (includeCalendarSnapshot) {
    files.push({name: "calendar.ics", bytes: utf8(calendarSnapshot(event.app))})
  }
  var date = createdAt.slice(0, 10)
  event.response
    .header()
    .set(
      "Content-Disposition",
      'attachment; filename="balaur-household-' + date + '.zip"',
    )
  return event.blob(200, "application/zip", zip(files))
}

module.exports = {exportArchive: exportArchive}
