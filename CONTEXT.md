# Shared Household

This context defines the shared information and private interactions for a household.

## Language

**Household**:
A group of Household Members who share one Household State.

**Household Member**:
A person who belongs to a Household and has private Conversations.
_Avoid_: User

**Household Administrator**:
A Household Member who manages membership and Calendar Connections.
_Avoid_: Admin, owner

**Household Invitation**:
A one-time invitation that lets a person become a Household Member.
_Avoid_: Signup link, invite code

**Household Server**:
The self-hosted server that owns the data of one Household. Each Household has
one Household Server.
_Avoid_: Backend, PocketBase instance

**Household Pairing**:
The process that connects one client to a Household Server and one Household
Member identity.
_Avoid_: Server setup, login

**Household Archive**:
A portable export of Household State and shared files. It excludes active
credentials and private Conversations.
_Avoid_: Backup, database dump

**Calendar Source**:
An external calendar that supplies schedule entries to a Household. Google
Calendar is one Calendar Source.
_Avoid_: Calendar backend, Google Calendar connection

**Calendar Connection**:
The Household-owned authorization and configuration for one Calendar Source.
A Household has at most one active Calendar Connection.
_Avoid_: Google token, integration

**Calendar Entry**:
A scheduled item supplied by a Calendar Source.
_Avoid_: Event

**Household Time Zone**:
The time zone that all Household Members use to view Calendar Entries.
_Avoid_: Device time zone, local time zone

**Household State**:
The authoritative records shared by all Household Members. It excludes Conversations and agent-generated summaries.
_Avoid_: Memory, shared memory

**Household Change**:
An attributed change to Household State. A Household Change keeps enough prior state to support a safe undo action.
_Avoid_: Event, mutation

**Conversation**:
A private message history between one Household Member and an agent. A Conversation can use Household State without becoming part of it.
_Avoid_: Shared conversation, memory

**Conversation Summary**:
A compact account of older Conversation turns. It provides context but never replaces authoritative Household State or original messages.
_Avoid_: Memory, Household State

**Shopping List**:
The one active collection of Shopping Items in a Household.
_Avoid_: Grocery list

**Shopping Item**:
A requested item in the Shopping List. It is pending, completed, or removed.
_Avoid_: Product
