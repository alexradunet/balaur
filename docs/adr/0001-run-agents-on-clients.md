# Run agents on clients

Each Flutter client runs its own Balaur agent and uses member-specific model settings. A shared backend owns Household State and private Conversations. This design keeps agent behavior portable while each client uses the same household data.
