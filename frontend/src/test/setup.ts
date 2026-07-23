import '@testing-library/jest-dom'

// jsdom lacks pointer-capture / scrollIntoView APIs that Radix (DropdownMenu,
// Dialog 등) 가 호출한다. 없으면 Radix 상호작용 테스트가 크래시하므로 안전하게 폴리필.
if (!Element.prototype.hasPointerCapture) {
  Element.prototype.hasPointerCapture = () => false
}
if (!Element.prototype.setPointerCapture) {
  Element.prototype.setPointerCapture = () => {}
}
if (!Element.prototype.releasePointerCapture) {
  Element.prototype.releasePointerCapture = () => {}
}
if (!Element.prototype.scrollIntoView) {
  Element.prototype.scrollIntoView = () => {}
}
