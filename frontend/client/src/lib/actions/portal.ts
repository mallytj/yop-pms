export function portal(node: HTMLElement, target: string = "body") {
  const targetEl = document.querySelector(target)!;
  targetEl.appendChild(node);
  return {
    destroy() {
      node.remove();
    },
  };
}
