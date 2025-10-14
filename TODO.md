# TODO: Fix Issues and Enhance Frontend for Travel Planner

## Backend Modifications
- [x] Update /api/route handler to prompt AI for JSON response with structured itinerary including per-day expenses
- [x] Parse AI response as JSON and return structured data instead of raw text

## Frontend Enhancements
- [x] Update App.vue to expect structured response data
- [x] Add loading state during API call
- [x] Display source and destination prominently
- [x] Render per-day expenses in a formatted list
- [x] Improve error handling and user feedback

## Testing
- [x] Run backend server
- [x] Run frontend development server
- [x] Test the full flow: input source/dest/budget, submit, verify structured display of per-day expenses
