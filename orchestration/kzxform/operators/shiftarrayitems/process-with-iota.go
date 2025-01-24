package shiftarrayitems

//func processShiftArray(data []byte, sourceRef operators.JsonReference, itemKTransformation *kazaam.Kazaam, params OperatorParams) ([]byte, error) {
//	const semLogContext = OperatorSemLogContext + "::process-nested-array"
//
//	sourceArray, err := operators.GetJsonArray(data, sourceRef)
//	if err != nil {
//		log.Error().Err(err).Msg(semLogContext)
//		return nil, err
//	}
//
//	lenOfSourceArray, err := operators.LenOfArray(sourceArray, operators.JsonReference{})
//	if err != nil {
//		log.Error().Err(err).Msg(semLogContext)
//		return nil, err
//	}
//
//	log.Info().Int("len", lenOfSourceArray).Msg(semLogContext)
//
//	// Variables to build new array
//	arrayItemNdx := 0
//	resultArray := []byte(`{}`)
//	resultLen := 0
//	var loopErr error
//	_, err = jsonparser.ArrayEach(sourceArray, func(value []byte, dataType jsonparser.ValueType, offset int, errParam error) {
//		if loopErr != nil {
//			log.Error().Err(err).Msg(semLogContext + " previous error in for-each")
//			return
//		}
//
//		itemTransformed := string(value)
//		accepted := true
//		if len(params.criteria) > 0 {
//			accepted, err = params.criteria.IsAccepted(value, map[string]interface{}{operators.CriterionSystemVariableKazaamArrayLen: lenOfSourceArray, operators.CriterionSystemVariableKazaamArrayLoopIndex: arrayItemNdx}) // isAccepted(value, criteria)
//		}
//		if accepted {
//			itemTransformed, err = itemKTransformation.TransformJSONStringToString(itemTransformed)
//		}
//		if err != nil {
//			log.Error().Err(err).Msg(semLogContext)
//			loopErr = err
//			return
//		}
//
//		if accepted || !params.filterItems {
//			resultArray, err = jsonparser.Set(resultArray, []byte(itemTransformed), OperatorsTempResultPropertyName, "[+]")
//			if err != nil {
//				loopErr = err
//				log.Error().Err(err).Msg(semLogContext)
//				return
//			}
//			resultLen++
//		}
//		arrayItemNdx++
//
//	})
//
//	if loopErr != nil {
//		return nil, loopErr
//	}
//
//	var rcItem []byte
//	switch resultLen {
//	case 0:
//		if params.flatten {
//			rcItem = []byte(`{}`)
//		} else {
//			rcItem = []byte(`[]`)
//		}
//	case 1:
//		if params.flatten {
//			var item0 []byte
//			var dt jsonparser.ValueType
//			item0, dt, _, err = jsonparser.Get(resultArray, OperatorsTempResultPropertyName, "[0]")
//			log.Trace().Str("dt", dt.String()).Msg(semLogContext)
//			rcItem = []byte(`{}`)
//			err = jsonparser.ObjectEach(item0, func(key, value []byte, dataType jsonparser.ValueType, offset int) error {
//
//				if dataType == jsonparser.String {
//					value = []byte(fmt.Sprintf(`"%s"`, string(value)))
//				}
//				rcItem, err = jsonparser.Set(rcItem, value, string(key))
//				return nil
//			})
//			if err != nil {
//				log.Error().Err(err).Msg(semLogContext)
//				return nil, err
//			}
//		} else {
//			var dt jsonparser.ValueType
//			rcItem, dt, _, err = jsonparser.Get(resultArray, OperatorsTempResultPropertyName)
//			if err != nil {
//				// Note: how to signal back an error?
//				log.Error().Err(err).Msg(semLogContext)
//				return nil, err
//			}
//			log.Info().Interface("data-type", dt).Msg(semLogContext)
//		}
//	default:
//		var dt jsonparser.ValueType
//		rcItem, dt, _, err = jsonparser.Get(resultArray, OperatorsTempResultPropertyName)
//		if err != nil {
//			// Note: how to signal back an error?
//			log.Error().Err(err).Msg(semLogContext)
//			return nil, err
//		}
//		log.Info().Interface("data-type", dt).Msg(semLogContext)
//	}
//
//	return rcItem, nil
//}
