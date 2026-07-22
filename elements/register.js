import { BalaurAddMenuElement } from "./add-menu.js";
import { BalaurComponentCardElement } from "./component-card.js";
import { BalaurDialogFrameElement } from "./dialog-frame.js";
import { BalaurInspectorElement } from "./inspector.js";
import { BalaurTaskListElement } from "./task-list.js";
import { BalaurWidgetFrameElement } from "./widget-frame.js";
import { BalaurWorkspaceNavElement } from "./workspace-nav.js";
import { defineElement } from "./element-utils.js";

defineElement("balaur-add-menu", BalaurAddMenuElement);
defineElement("balaur-component-card", BalaurComponentCardElement);
defineElement("balaur-dialog-frame", BalaurDialogFrameElement);
defineElement("balaur-inspector", BalaurInspectorElement);
defineElement("balaur-task-list", BalaurTaskListElement);
defineElement("balaur-widget-frame", BalaurWidgetFrameElement);
defineElement("balaur-workspace-nav", BalaurWorkspaceNavElement);
