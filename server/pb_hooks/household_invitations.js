var householdAdministratorRole = "household_administrator"
var invitationCollection = "household_invitations"
var invitationLifetimeMilliseconds = 24 * 60 * 60 * 1000
var invitationValuePattern = /^[A-Za-z0-9]{48}$/

function noStore(event) {
  event.response.header().set("Cache-Control", "no-store")
  event.response.header().set("X-Content-Type-Options", "nosniff")
}

function requireHouseholdAdministrator(event) {
  if (!event.auth || event.auth.getString("role") !== householdAdministratorRole) {
    throw new ForbiddenError("Only a Household Administrator can manage Household Invitations.")
  }
  return event.auth
}

function invitationInput(event) {
  var body = event.requestInfo().body
  return {
    role: String(body.role || ""),
  }
}

function redemptionInput(event) {
  var body = event.requestInfo().body
  return {
    invitation: String(body.invitation || ""),
    displayName: String(body.displayName || "").trim(),
    email: String(body.email || "").trim().toLowerCase(),
    password: String(body.password || ""),
    passwordConfirm: String(body.passwordConfirm || ""),
  }
}

function requireInvitationRole(role) {
  if (role !== "member" && role !== householdAdministratorRole) {
    throw new BadRequestError("Select a valid Household Member role.")
  }
}

function validateRedemptionInput(input) {
  if (!invitationValuePattern.test(input.invitation)) {
    throw new NotFoundError("The Household Invitation is invalid.")
  }
  if (input.displayName.length < 1 || input.displayName.length > 120) {
    throw new BadRequestError("Enter a valid name.")
  }
  if (input.email.length > 254 || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(input.email)) {
    throw new BadRequestError("Enter a valid email address.")
  }
  if (input.password.length < 8 || input.password.length > 71) {
    throw new BadRequestError("Use a password with 8 to 71 characters.")
  }
  if (input.password !== input.passwordConfirm) {
    throw new BadRequestError("The passwords do not match.")
  }
}

function expiresAtMilliseconds(invitation) {
  return new Date(invitation.getString("expiresAt")).getTime()
}

function publicState(invitation) {
  var state = invitation.getString("state")
  if (state === "active" && expiresAtMilliseconds(invitation) <= Date.now()) {
    return "expired"
  }
  return state
}

function publicInvitation(app, invitation) {
  var creator = app.findRecordById("members", invitation.getString("creator"))
  return {
    id: invitation.id,
    creator: {
      id: creator.id,
      displayName: creator.getString("displayName"),
    },
    expiresAt: invitation.getString("expiresAt"),
    role: invitation.getString("role"),
    state: publicState(invitation),
  }
}

function createInvitation(event) {
  noStore(event)
  var creator = requireHouseholdAdministrator(event)
  var input = invitationInput(event)
  requireInvitationRole(input.role)

  var value = $security.randomString(48)
  var invitations = event.app.findCollectionByNameOrId(invitationCollection)
  var invitation = new Record(invitations)
  invitation.set("tokenHash", $security.sha256(value))
  invitation.set("creator", creator.id)
  invitation.set(
    "expiresAt",
    new Date(Date.now() + invitationLifetimeMilliseconds).toISOString(),
  )
  invitation.set("role", input.role)
  invitation.set("state", "active")
  event.app.save(invitation)

  return event.json(201, {
    invitation: publicInvitation(event.app, invitation),
    value: value,
  })
}

function listInvitations(event) {
  noStore(event)
  requireHouseholdAdministrator(event)
  var records = event.app.findAllRecords(invitationCollection)
  records.sort(function (left, right) {
    return right.getString("created").localeCompare(left.getString("created"))
  })
  return event.json(200, {
    items: records.map(function (record) {
      return publicInvitation(event.app, record)
    }),
  })
}

function cancelInvitation(event) {
  noStore(event)
  requireHouseholdAdministrator(event)
  var publicResult
  event.app.runInTransaction(function (transactionApp) {
    var invitation = transactionApp.findRecordById(
      invitationCollection,
      event.request.pathValue("id"),
    )
    if (publicState(invitation) !== "active") {
      throw new ApiError(409, "The Household Invitation is not active.")
    }
    invitation.set("state", "canceled")
    transactionApp.save(invitation)
    publicResult = publicInvitation(transactionApp, invitation)
  })
  return event.json(200, {
    invitation: publicResult,
  })
}

function redeemInvitation(event) {
  noStore(event)
  var input = redemptionInput(event)
  validateRedemptionInput(input)
  var tokenHash = $security.sha256(input.invitation)
  var member

  event.app.runInTransaction(function (transactionApp) {
    var invitation
    try {
      invitation = transactionApp.findFirstRecordByData(
        invitationCollection,
        "tokenHash",
        tokenHash,
      )
    } catch (error) {
      throw new NotFoundError("The Household Invitation is invalid.")
    }

    var state = invitation.getString("state")
    if (state === "used") {
      throw new ApiError(409, "The Household Invitation was already used.")
    }
    if (state === "canceled") {
      throw new ApiError(409, "The Household Invitation was canceled.")
    }
    if (expiresAtMilliseconds(invitation) <= Date.now()) {
      throw new ApiError(410, "The Household Invitation expired.")
    }

    var members = transactionApp.findCollectionByNameOrId("members")
    member = new Record(members)
    member.setEmail(input.email)
    member.setPassword(input.password)
    member.setVerified(true)
    member.set("displayName", input.displayName)
    member.set("role", invitation.getString("role"))
    transactionApp.save(member)

    invitation.set("state", "used")
    transactionApp.save(invitation)
  })

  event.auth = member
  return $apis.recordAuthResponse(event, member, "password", {})
}

module.exports = {
  cancelInvitation: cancelInvitation,
  createInvitation: createInvitation,
  listInvitations: listInvitations,
  redeemInvitation: redeemInvitation,
}
