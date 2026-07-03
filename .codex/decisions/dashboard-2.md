## dashboard - 2

### Primary Service
Monitoring Dashboard

### Secondary Services
- None

### Choice Made
Use a fixed **40% / 60%** two-column layout for the interface management section (table left, detail right) instead of a draggable split pane.

### Alternatives Considered
- Custom draggable `ResizableSplit` with mouse-driven width (original plan)
- Third-party panel library (e.g. `react-resizable-panels`)

### Pros
- No drag-handle edge cases (min/max width, pointer capture, SSR, touch)
- Less code to build and maintain
- Consistent layout across users and viewports

### Cons
- Users cannot widen the table or detail panel to taste
- Very long interface names or dense detail may feel cramped on the 40% side

### Cost / Benefit
Fixed CSS columns trade minor layout flexibility for substantially simpler implementation and fewer maintenance paths; stacks to a single column below 900px per existing responsive patterns.

### Status
Accepted
