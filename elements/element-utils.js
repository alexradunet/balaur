export function defineElement(name, constructor) {
  const registered = customElements.get(name);
  if (registered) return registered;
  customElements.define(name, constructor);
  return constructor;
}
