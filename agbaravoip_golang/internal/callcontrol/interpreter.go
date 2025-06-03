package callcontrol

import (
	"github.com/user/agbaravoip_golang/internal/domain" // Corrected import path
)

// ExecuteAgbaraXML iterates through a slice of CallControlElements and executes them sequentially.
// It processes the result of each execution to determine the next course of action,
// such as continuing to the next element, redirecting to a new XML document, or hanging up the call.
func ExecuteAgbaraXML(
	callCtx domain.MinimalCallContext, // Use MinimalCallContext from domain
	elements []domain.CallControlElement,
	eslConn domain.EslConnectionExecutor, // This is now directly available via callCtx if needed, or pass explicitly
	callSvc domain.CallServicerForESL,
) domain.CallControlResult {
	currentResult := domain.CallControlResult{Action: domain.ActionContinue}

	for _, element := range elements {
		if callCtx.IsHangupInitiated() {
			callCtx.Log().Info("Hangup initiated by ESL event or previous critical error, skipping further XML element execution.")
			if currentResult.Action != domain.ActionHangup { // Ensure final action is Hangup
				currentResult.Action = domain.ActionHangup
				currentResult.Err = nil
			}
			break
		}

		callCtx.Log().Infof("Executing XML element: %T", element)
		// Pass eslConn from CallContext or ensure element.Execute gets it if it's not the one in CallContext.
		// The eslConn parameter to ExecuteAgbaraXML is the one that should be passed to element.Execute.

		result := element.Execute(callCtx, eslConn, callSvc)
		currentResult = result

		if result.Err != nil {
			callCtx.Log().Errorf("Error executing element %T: %v. Setting action to Hangup.", element, result.Err)
			currentResult.Action = domain.ActionHangup
			break
		}

		if result.Action == domain.ActionHangup || result.Action == domain.ActionRedirect {
			callCtx.Log().Infof("Element %T returned action: %s. Stopping processing of current XML document.", element, result.Action)
			break
		}
		// If ActionContinue, loop proceeds.
		// Other actions like Gather/Record might return ActionContinue if subsequent elements in the *same* doc are to be processed.
	}

	// If loop completed and last action was Continue, but call was remotely disconnected during the *last* element execution.
	if currentResult.Action == domain.ActionContinue && callCtx.IsHangupInitiated() {
	    callCtx.Log().Info("Call was hung up during XML execution (possibly during the last element), ensuring final action is Hangup.")
	    currentResult.Action = domain.ActionHangup
	    currentResult.Err = nil
	}

	callCtx.Log().Infof("Finished executing AgbaraXML elements. Final determined action: %s", currentResult.Action)
	if currentResult.Action == domain.ActionRedirect {
		callCtx.Log().Infof("Redirecting to URL: %s, Method: %s", currentResult.RedirectURL, currentResult.RedirectMethod)
	}
	if currentResult.Err != nil {
		callCtx.Log().Errorf("Error at end of AgbaraXML execution: %v", currentResult.Err)
	}

	return currentResult
}
