// Package constants provides application-wide constant values for the gitbak application.
//
// This package centralizes constant values that are used throughout the application,
// making them easy to maintain and update. It includes visual elements like ASCII art,
// as well as other fixed values that define the application's behavior and appearance.
//
// # Core Components
//
// - Logo: ASCII art representation of the gitbak logo
// - Tagline: The application's tagline/motto
//
// # Usage
//
// The constants in this package can be imported and used directly:
//
//	import "github.com/bashhack/gitbak/internal/constants"
//
//	func displayLogo() {
//	    fmt.Println(constants.Logo)
//	    fmt.Println(constants.Tagline)
//	}
//
// # Design Considerations
//
// Constants are grouped in this package for several reasons:
//
// - Centralization: Makes it easy to find and update application-wide constants
// - Discoverability: Provides a clear location for all fixed values
// - Consistency: Ensures the same values are used throughout the application
// - Separation of Concerns: Keeps presentation elements separate from business logic
//
// # Maintenance
//
// When adding new constants to this package:
//
// - Group related constants together
// - Provide clear documentation for each constant
// - Consider the scope and usage across the application
package constants