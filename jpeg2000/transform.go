package jpeg2000

/*
type transformedTile struct {
	left   uint32
	top    uint32
	width  uint32
	height uint32
	items  []float64
}

type subbandCoefficients struct {
	width  uint32
	height uint32
	items  []float64
}

type transform interface {
	//calculate(coefficients []*subbandCoefficients, width, height uint32)
	iterate()
}

func (tile *tile) transformTile(header *Header, c uint16) *transformedTile {
	component := tile.components[c]
	codingStyleParameters := component.codingStyleParameters
	quantizationParameters := component.quantizationParameters
	decompositionLevelsCount := codingStyleParameters.NumberOfLevels
	spqcds := quantizationParameters.SPqcd                           // SPqcds
	scalarExpounded := quantizationParameters.QuantizationStyle != 1 // scalarExpounded
	guardBits := quantizationParameters.GuardBits
	segmentationSymbolUsed := codingStyleParameters.SegmentationSymbols
	precision := header.Components[c].BitDepth

	reversible := codingStyleParameters.ReversableFilter
	var transform transform

	if reversible {
		transform = newReversibleTransform()
	} else {
		transform = newIrreversibleTransform()
	}

	var subbandCoefficients []*subbandCoefficients
	b := 0

	for i := uint16(0); i <= uint16(decompositionLevelsCount); i++ {
		resolution := component.resolutions[i]

		width := resolution.trx1 - resolution.trx0
		height := resolution.try1 - resolution.try0

		// Allocate space for the whole sublevel.
		var coefficients []float64
		coefficients = make([]float64, width*height)

		for j := 0; j < len(resolution.subbands); j++ {
			var mu, epsilon uint16

			if !scalarExpounded {
				// formula E-5
				mu = spqcds[0].Mantissa      //mu
				epsilon = spqcds[0].Exponent // epsilon

				if i > 0 {
					epsilon += 1 - i
				}
			} else {
				mu = spqcds[b].Mantissa      //mu
				epsilon = spqcds[b].Exponent //epsilon
				b++
			}

			subband := resolution.subbands[j]
			gainLog2 := SubbandsGainLog2[subband.subbandType]

			// calculate quantization coefficient (Section E.1.1.1)
			var delta float64
			if reversible {
				delta = 1
			} else {
				delta = math.Pow(2, precision+gainLog2-epsilon) * (1 + mu/2048)
			}
			mb := guardBits + epsilon - 1

			// In the first resolution level, copyCoefficients will fill the
			// whole array with coefficients. In the succeeding passes,
			// copyCoefficients will consecutively fill in the values that belong
			// to the interleaved positions of the HL, LH, and HH coefficients.
			// The LL coefficients will then be interleaved in Transform.iterate().
			copyCoefficients(coefficients, width, height, subband, delta, mb, reversible, segmentationSymbolUsed)
		}

		subbandCoefficients = append(subbandCoefficients, &subbandCoefficients{width: width, height: height, items: coefficients})
	}

	result := transform.calculate(subbandCoefficients, component.tcx0, component.tcy0)
	return &transformedTile{left: component.tcx0, top: component.tcy0, width: result.width, height: result.height, items: result.items}
}
*/
type transformedComponent struct {
	left   uint32
	top    uint32
	width  uint32
	height uint32
	items  []float64
}

func (header *Header) transformComponents() []*transformedComponent {
	var resultImages []*transformedComponent
	/*
		components := header.Components
		componentsCount := header.Size.Csiz

		for i := 0; i < len(header.tiles); i++ {
			tile := header.tiles[i]
			var transformedTiles []*transformedTile

			for c := uint16(0); c < componentsCount; c++ {
				transformedTiles = append(transformedTiles, tile.transformTile(header, c))
			}
			var result transformedComponent
			var tile0 = transformedTiles[0]

			out := make([]float64, len(tile0.items)*int(componentsCount))

			result.left = tile0.left
			result.top = tile0.top
			result.width = tile0.width
			result.height = tile0.height
			result.items = out

			// Section G.2.2 Inverse multi component transform
			var shift, offset uint32
			var pos, y0, y1, y2 uint32

			if tile.codingStyleDefaultParameters.MultipleComponentTransformation {
				var y0items, y1items, y2items, y3items []float64
				fourComponents := componentsCount == 4
				y0items = transformedTiles[0].items
				y1items = transformedTiles[1].items
				y2items = transformedTiles[2].items

				if fourComponents {
					y3items = transformedTiles[3].items
				}

				// HACK: The multiple component transform formulas below assume that
				// all components have the same precision. With this in mind, we
				// compute shift and offset only once.
				shift = components[0].BitDepth - 8
				offset = (128 << shift) + 0.5

				var component0 = tile.components[0]
				var alpha01 = componentsCount - 3

				if !component0.codingStyleParameters.ReversableFilter {
					// inverse irreversible multiple component transform
					for j := 0; j < len(y0items); j++ {
						y0 = y0items[j] + offset
						y1 = y1items[j]
						y2 = y2items[j]
						out[pos] = (y0 + 1.402*y2) >> shift
						pos++
						out[pos] = (y0 - 0.34413*y1 - 0.71414*y2) >> shift
						pos++
						out[pos] = (y0 + 1.772*y1) >> shift
						pos++

						pos += alpha01
					}
				} else {
					// inverse reversible multiple component transform
					for j = 0; j < len(y0items); j++ {
						y0 = y0items[j] + offset
						y1 = y1items[j]
						y2 = y2items[j]
						g := y0 - ((y2 + y1) >> 2)

						out[pos] = (g + y2) >> shift
						pos++
						out[pos] = g >> shift
						pos++
						out[pos] = (g + y1) >> shift
						pos++

						pos += alpha01
					}
				}
				if fourComponents {
					pos = 3

					for j = 0; j < len(y0items); j++ {
						out[pos] = (y3items[j] + offset) >> shift

						pos += 4
					}
				}
			} else {
				// no multi-component transform
				for c = 0; c < componentsCount; c++ {
					items := transformedTiles[c].items
					shift = components[c].precision - 8
					offset = (128 << shift) + 0.5
					pos = c
					for j = 0; j < len(items); j++ {
						out[pos] = (items[j] + offset) >> shift
						pos += componentsCount
					}
				}
			}

			resultImages = append(resultImages, &result)
		}
	*/
	return resultImages
}

/*
type baseTransform struct {
}

func (tform *baseTransform) iterate(ll *subbandCoefficients, hl_lh_hh *subbandCoefficients, u0, v0 uint32) *subbandCoefficients {
	llWidth := ll.width
	llHeight := ll.height
	llItems := ll.items

	width := hl_lh_hh.width
	height := hl_lh_hh.height
	items := hl_lh_hh.items
	var i, j, k, l, u, v int

	// Interleave LL according to Section F.3.3
	for i = 0; i < llHeight; i++ {
		l = i * 2 * width

		for j = 0; j < llWidth; j++ {
			items[l] = llItems[k]

			k++
			l += 2
		}
	}
	// The LL band is not needed anymore.
	llItems = nil
	ll.items = nil

	var bufferPadding = 4
	rowBuffer := make([]float64, width+2*bufferPadding)

	// Section F.3.4 HOR_SR
	if width == 1 {
		// if width = 1, when u0 even keep items as is, when odd divide by 2
		if (u0 & 1) != 0 {
			k = 0
			for v = 0; v < height; v++ {
				items[k] *= 0.5
				k += width
			}
		}
	} else {
		k = 0
		for v = 0; v < height; v++ {
			rowBuffer.set(items.subarray(k, k+width), bufferPadding)

			this.extend(rowBuffer, bufferPadding, width)
			this.filter(rowBuffer, bufferPadding, width)

			items.set(rowBuffer.subarray(bufferPadding, bufferPadding+width), k)

			k += width
		}
	}

	// Accesses to the items array can take long, because it may not fit into
	// CPU cache and has to be fetched from main memory. Since subsequent
	// accesses to the items array are not local when reading columns, we
	// have a cache miss every time. To reduce cache misses, get up to
	// 'numBuffers' items at a time and store them into the individual
	// buffers. The colBuffers should be small enough to fit into CPU cache.
	numBuffers := 16
	var colBuffers [][]float64

	for i = 0; i < numBuffers; i++ {
		colBuffers = append(colBuffers, make([]float64, height+2*bufferPadding))
	}
	var b int
	currentBuffer := 0
	ll = bufferPadding + height

	// Section F.3.5 VER_SR
	if height == 1 {
		// if height = 1, when v0 even keep items as is, when odd divide by 2
		if (v0 & 1) != 0 {
			for u = 0; u < width; u++ {
				items[u] *= 0.5
			}
		}
	} else {
		for u = 0; u < width; u++ {
			// if we ran out of buffers, copy several image columns at once
			if currentBuffer == 0 {
				numBuffers = math.Min(width-u, numBuffers)

				k = u
				for l = bufferPadding; l < ll; l++ {
					for b = 0; b < numBuffers; b++ {
						colBuffers[b][l] = items[k+b]
					}

					k += width
				}

				currentBuffer = numBuffers
			}

			currentBuffer--
			buffer := colBuffers[currentBuffer]
			this.extend(buffer, bufferPadding, height)
			this.filter(buffer, bufferPadding, height)

			// If this is last buffer in this group of buffers, flush all buffers.
			if currentBuffer == 0 {
				k = u - numBuffers + 1

				for l = bufferPadding; l < ll; l++ {
					for b = 0; b < numBuffers; b++ {
						items[k+b] = colBuffers[b][l]
					}

					k += width
				}
			}
		}
	}

	return subbandCoefficients{width: width, height: height, items: items}
}

func extendTransform(tform transform, buffer []float64, offset, size int) {
	// Section F.3.7 extending... using max extension of 4
	i1 := offset - 1
	j1 := offset + 1
	i2 := offset + size - 2
	j2 := offset + size

	buffer[i1] = buffer[j1]
	i1--
	j1++

	buffer[j2] = buffer[i2]
	j2++
	i2--

	buffer[i1] = buffer[j1]
	i1--
	j1++

	buffer[j2] = buffer[i2]
	j2++
	i2--

	buffer[i1] = buffer[j1]
	i1--
	j1++

	buffer[j2] = buffer[i2]
	j2++
	i2--

	buffer[i1] = buffer[j1]
	buffer[j2] = buffer[i2]
}

func calculateTransform(tform transform, subbands []*subbandCoefficients, u0, v0 uint32) *subbandCoefficients {
	ll := subbands[0]

	for i := 1; i < len(subbands); i++ {
		ll = iterateTransform(tform, ll, subbands[i], u0, v0)
	}

	return ll
}
*/
