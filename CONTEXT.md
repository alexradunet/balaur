# Shared Household

This context defines the shared information and private interactions for a household.

## Language

**Household**:
A group of Household Members who share one Household State.

**Household Member**:
A person who belongs to a Household and has private Conversations.
_Avoid_: User

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
