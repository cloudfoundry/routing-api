package models_test

import (
	. "github.com/cloudfoundry-incubator/routing-api/models"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Models", func() {
	Describe("RouterGroup", func() {
		var rg RouterGroup

		Describe("Validate", func() {
			It("succeeds for valid router group", func() {
				rg = RouterGroup{
					Name:            "router-group-1",
					Type:            "tcp",
					ReservablePorts: "10-20",
				}
				err := rg.Validate()
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails for missing type", func() {
				rg = RouterGroup{
					Name:            "router-group-1",
					ReservablePorts: "10-20",
				}
				err := rg.Validate()
				Expect(err).To(HaveOccurred())
			})

			It("fails for missing name", func() {
				rg = RouterGroup{
					Type:            "tcp",
					ReservablePorts: "10-20",
				}
				err := rg.Validate()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ReservablePorts", func() {
		var ports ReservablePorts

		Describe("Validate", func() {

			It("succeeds for valid reservable ports", func() {
				ports = "1,5,10-20,21-30"
				err := ports.Validate()
				Expect(err).NotTo(HaveOccurred())
			})

			It("fails for overlapping ranges", func() {
				ports = "10-20,20-30"
				err := ports.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Overlapping values: [10-20] and [20-30]"))
			})

			It("fails for overlapping values", func() {
				ports = "1,1,2,3,3,4"
				err := ports.Validate()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Overlapping values: 1 and 1"))
			})

			It("fails for invalid reservable ports", func() {
				ports = "foo!"
				err := ports.Validate()
				Expect(err).To(HaveOccurred())
			})
		})

		Describe("Parse", func() {
			It("validates a single unsigned integer", func() {
				ports = "9999"
				r, err := ports.Parse()
				Expect(err).NotTo(HaveOccurred())

				Expect(len(r)).To(Equal(1))
				start, end := r[0].Endpoints()
				Expect(start).To(Equal(uint64(9999)))
				Expect(end).To(Equal(uint64(9999)))
			})

			It("validates multiple integers", func() {
				ports = "9999,1111,2222"
				r, err := ports.Parse()
				Expect(err).NotTo(HaveOccurred())
				Expect(len(r)).To(Equal(3))

				expected := []uint64{9999, 1111, 2222}
				for i := 0; i < len(r); i++ {
					start, end := r[i].Endpoints()
					Expect(start).To(Equal(expected[i]))
					Expect(end).To(Equal(expected[i]))
				}
			})

			It("validates a range", func() {
				ports = "1-10"
				r, err := ports.Parse()
				Expect(err).NotTo(HaveOccurred())

				Expect(len(r)).To(Equal(1))
				start, end := r[0].Endpoints()
				Expect(start).To(Equal(uint64(1)))
				Expect(end).To(Equal(uint64(10)))
			})

			It("validates a list of ranges and integers", func() {
				ports = "1-10,20-22,45,50-60"
				r, err := ports.Parse()
				Expect(err).NotTo(HaveOccurred())

				Expect(len(r)).To(Equal(4))
				expected := []uint64{1, 10, 20, 22, 45, 45, 50, 60}
				for i := 0; i < len(r); i++ {
					start, end := r[i].Endpoints()
					Expect(start).To(Equal(expected[2*i]))
					Expect(end).To(Equal(expected[2*i+1]))
				}
			})

			It("errors on range with 3 dashes", func() {
				ports = "10-999-1000"
				_, err := ports.Parse()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("range (10-999-1000) has too many '-' separators"))
			})

			It("errors on a negative integer", func() {
				ports = "-9999"
				_, err := ports.Parse()
				Expect(err).To(HaveOccurred())
			})

			It("errors on a incomplete range", func() {
				ports = "1-"
				_, err := ports.Parse()
				Expect(err).To(HaveOccurred())
			})

			It("errors on non-numeric input", func() {
				ports = "adsfasdf"
				_, err := ports.Parse()
				Expect(err).To(HaveOccurred())
			})

			It("errors when range starts with lower number", func() {
				ports = "10000-9999"
				_, err := ports.Parse()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("range (10000-9999) must be in ascending numeric order"))
			})
		})
	})

	Describe("Range", func() {
		Describe("Overlaps", func() {
			testRange := NewRange(10, 20)

			It("validates non-overlapping ranges", func() {
				r := NewRange(21, 30)
				Expect(testRange.Overlaps(r)).To(BeFalse())
			})

			It("finds overlapping ranges of single values", func() {
				r1 := NewRange(10, 10)
				r2 := NewRange(10, 10)
				Expect(r1.Overlaps(r2)).To(BeTrue())
			})

			It("finds overlapping ranges of single value and range", func() {
				r2 := NewRange(15, 15)
				Expect(testRange.Overlaps(r2)).To(BeTrue())
			})

			It("finds overlapping ranges of single value upper bound and range", func() {
				r2 := NewRange(20, 20)
				Expect(testRange.Overlaps(r2)).To(BeTrue())
			})

			It("validates single value one above upper bound range", func() {
				r2 := NewRange(21, 21)
				Expect(testRange.Overlaps(r2)).To(BeFalse())
			})

			It("finds overlapping ranges when start overlaps", func() {
				r := NewRange(15, 30)
				Expect(testRange.Overlaps(r)).To(BeTrue())
			})

			It("finds overlapping ranges when end overlaps", func() {
				r := NewRange(5, 15)
				Expect(testRange.Overlaps(r)).To(BeTrue())
			})

			It("finds overlapping ranges when the range is a superset", func() {
				r := NewRange(9, 21)
				Expect(testRange.Overlaps(r)).To(BeTrue())
			})
		})
	})
})
