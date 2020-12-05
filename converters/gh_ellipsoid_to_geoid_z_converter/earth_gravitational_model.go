package gh_ellipsoid_to_geoid_z_converter

import (
	"bufio"
	"github.com/mfbonfigli/gocesiumtiler/utils"
	"log"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
)

const sqrt03 = 1.7320508075688772935274463415059
const sqrt05 = 2.2360679774997896964091736687313
const sqrt13 = 3.6055512754639892931192212674705
const sqrt17 = 4.1231056256176605498214098559741
const sqrt21 = 4.5825756949558400065880471937280
const defaultOrder = 180

type egm struct {
	wgs84                    bool
	nmax                     int
	semiMajor                float64
	esq                      float64
	c2                       float64
	rkm                      float64
	grava                    float64
	star                     float64
	cnmGeopCoef, snmGeopCoef []float64
	aClenshav, bClenshaw, as []float64
}

// Inits a new earth gravitational model according to the default parameters
func newDefaultEarthGravitationalModel() *egm {
	return newEarthGraviationalModel(defaultOrder, true)
}

func newEarthGraviationalModel(nmax int, wgs84 bool) *egm {
	model := egm{
		nmax:  nmax,
		wgs84: wgs84,
	}
	if wgs84 {
		model.semiMajor = 6378137.0
		model.esq = 0.00669437999013
		model.c2 = 108262.9989050e-8
		model.rkm = 3.986004418e+14
		model.grava = 9.7803267714
		model.star = 0.001931851386
	} else {
		model.semiMajor = 6378135.0
		model.esq = 0.006694317778
		model.c2 = 108263.0e-8
		model.rkm = 3.986005e+14
		model.grava = 9.7803327
		model.star = 0.005278994
	}
	cleanshawLength := locatingArray(nmax + 3)
	geopCoefLength := locatingArray(nmax + 1)
	model.aClenshav = make([]float64, cleanshawLength)
	model.bClenshaw = make([]float64, cleanshawLength)
	model.cnmGeopCoef = make([]float64, geopCoefLength)
	model.snmGeopCoef = make([]float64, geopCoefLength)
	model.as = make([]float64, nmax+1)

	exPath := utils.GetExecutablePath()

	// Loading Earth Gravitational Model data
	err := model.load(path.Join(exPath, "static","egm180.nor"))
	if err != nil {
		log.Fatal("error loading gravitational model data", err)
	}

	return &model
}

func locatingArray(n int) int {
	return ((n + 1) * n) >> 1
}

func (egm *egm) load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, " ")
		i := 0
		for _, token := range tokens {
			if token != "" && token != " " {
				// copy and increment index
				tokens[i] = token
				i++
			}
		}
		tokens = tokens[:i]
		n, err := strconv.Atoi(tokens[0])
		if err != nil {
			return err
		}
		m, err := strconv.Atoi(tokens[1])
		if err != nil {
			return err
		}
		cbar, err := strconv.ParseFloat(tokens[2], 64)
		if err != nil {
			return err
		}
		sbar, err := strconv.ParseFloat(tokens[3], 64)
		if err != nil {
			return err
		}
		if n < egm.nmax {
			ll := locatingArray(n) + m
			egm.cnmGeopCoef[ll] = cbar
			egm.snmGeopCoef[ll] = sbar
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	egm.initialize()
	return nil
}

func (egm *egm) initialize() {
	if egm.wgs84 {
		c2n := make([]float64, 6)
		c2n[1] = egm.c2
		sign := 1
		esqi := egm.esq
		for i := 2; i < len(c2n); i++ {
			sign *= -1
			esqi *= egm.esq
			c2n[i] = float64(sign) * (3 * esqi) / ((2*float64(i) + 1) * (2*float64(i) + 3)) * (1 - float64(i) + (5 * float64(i) * egm.c2 / egm.esq))
		}
		egm.cnmGeopCoef[3] += c2n[1] / sqrt05
		egm.cnmGeopCoef[10] += c2n[2] / 3
		egm.cnmGeopCoef[21] += c2n[3] / sqrt13
		if egm.nmax > 6 {
			egm.cnmGeopCoef[36] += c2n[4] / sqrt17
		}
		if egm.nmax > 9 {
			egm.cnmGeopCoef[55] += c2n[5] / sqrt21
		}
	} else {
		egm.cnmGeopCoef[3] += 4.841732e-04
		egm.cnmGeopCoef[10] += -7.8305e-07
	}
	for i := 0; i < egm.nmax; i++ {
		egm.as[i] = -math.Sqrt(1.0 + 1.0/(2*(float64(i)+1)))
	}
	for i := 0; i <= egm.nmax; i++ {
		for j := i + 1; j < egm.nmax; j++ {
			ll := locatingArray(j) + i
			n := 2*j + 1
			ji := (j - i) * (j + i)
			egm.aClenshav[ll] = math.Sqrt(float64(n) * (2*float64(j) - 1) / float64(ji))
			egm.bClenshaw[ll] = math.Sqrt(float64(n) * float64(j+i-1) * float64(j-i-1) / float64(ji*(2*j-3)))
		}
	}
}

func (egm *egm) heightOffset(lon, lat, height float64) float64 {
	cr := make([]float64, egm.nmax+1)
	sr := make([]float64, egm.nmax+1)
	s11 := make([]float64, egm.nmax+3)
	s12 := make([]float64, egm.nmax+3)
	phi := lat / 180 * math.Pi
	sin_phi := math.Sin(phi)
	sin2_phi := sin_phi * sin_phi
	rni := math.Sqrt(1.0 - egm.esq*sin2_phi)
	rn := egm.semiMajor / rni
	t22 := (rn + height) * math.Cos(phi)
	x2y2 := t22 * t22
	z1 := ((rn * (1 - egm.esq)) + height) * sin_phi
	th := (math.Pi / 2.0) - math.Atan(z1/math.Sqrt(x2y2))
	y := math.Sin(th)
	t := math.Cos(th)
	f1 := egm.semiMajor / math.Sqrt(x2y2+z1*z1)
	f2 := f1 * f1
	rlam := lon / 180 * math.Pi
	var gravn float64
	if egm.wgs84 {
		gravn = egm.grava * (1.0 + egm.star*sin2_phi) / rni
	} else {
		gravn = egm.grava*(1.0+egm.star*sin2_phi) + 0.000023461*(sin2_phi*sin2_phi)
	}
	sr[0] = 0
	sr[1] = math.Sin(rlam)
	cr[0] = 1
	cr[1] = math.Cos(rlam)
	for j := 2; j <= egm.nmax; j++ {
		sr[j] = (2.0 * cr[1] * sr[j-1]) - sr[j-2]
		cr[j] = (2.0 * cr[1] * cr[j-1]) - cr[j-2]
	}
	var sht, previousSht float64 = 0, 0
	for i := egm.nmax; i >= 0; i-- {
		for j := egm.nmax; j >= i; j-- {
			ll := locatingArray(j) + i
			ll2 := ll + j + 1
			ll3 := ll2 + j + 2
			ta := egm.aClenshav[ll2] * f1 * t
			tb := egm.bClenshaw[ll3] * f2
			s11[j] = (ta * s11[j+1]) - (tb * s11[j+2]) + egm.cnmGeopCoef[ll]
			s12[j] = (ta * s12[j+1]) - (tb * s12[j+2]) + egm.snmGeopCoef[ll]
		}
		previousSht = sht
		sht = (-egm.as[i] * y * f1 * sht) + (s11[i] * cr[i]) + (s12[i] * sr[i])
	}
	return ((s11[0]+s12[0])*f1 + (previousSht * sqrt03 * y * f2)) * egm.rkm / (egm.semiMajor * (gravn - (height * 0.3086e-5)))
}
