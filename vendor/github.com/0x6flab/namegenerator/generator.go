package namegenerator

import (
	"math/rand"
	"sync"
	"time"
)

// NameGenerator is an interface for generating names.
type NameGenerator interface {

	// Generate generates a name based on the gender.
	//
	// Example:
	//  name := generator.Generate()
	//  fmt.Println(name)
	// Output:
	//  `John-Smith`
	Generate() string

	// GenerateNames generates a list of names.
	//
	// Example:
	//  names := generator.GenerateNames(10)
	//  fmt.Println(names)
	// Output:
	//  `[Dryke-Monroe Scarface-Lesway Shelden-Corsale Marcus-Ivett Victor-Nesrallah Merril-Gulick Leonardo-Lindler Maurits-Lias Rawley-Connor Elvis-Khouderchah]`
	GenerateNames(int) []string

	// WithGender generates a name based on the gender.
	//
	// Example:
	//  name := generator.Generate().WithGender("male")
	//  fmt.Println(name)
	// Output:
	//  `John-Smith`
	WithGender(string) NameGenerator
}

// nameGenerator is a struct that implements NameGenerator.
type nameGenerator struct {
	gender string
}

// NewNameGenerator returns a new NameGenerator.
//
// Example to generate general names:
//
//	generator := namegenerator.NewNameGenerator()
//
// Example to generate male names:
//
//	generator := namegenerator.NewNameGenerator().WithGender("male")
//
// Example to generate female names:
//
//	generator := namegenerator.NewNameGenerator().WithGender("female")
func NewNameGenerator() NameGenerator {
	return &nameGenerator{
		gender: "",
	}
}

func (namegen *nameGenerator) WithGender(gender string) NameGenerator {
	namegen.gender = gender

	return namegen
}

func (namegen *nameGenerator) Generate() string {
	frandom := rand.New(rand.NewSource(time.Now().UnixNano()))
	grandom := rand.New(rand.NewSource(time.Now().UnixNano()))

	randonFamilyName := FamilyNames[frandom.Intn(len(FamilyNames))]

	switch namegen.gender {
	case "male":
		randomMaleName := MaleNames[grandom.Intn(len(MaleNames))]

		return randomMaleName + "-" + randonFamilyName
	case "female":
		randomFemaleName := FemaleNames[grandom.Intn(len(FemaleNames))]

		return randomFemaleName + "-" + randonFamilyName
	default:
		randomName := GeneralNames[grandom.Intn(len(GeneralNames))]

		return randomName + "-" + randonFamilyName
	}
}

func (namegen *nameGenerator) GenerateNames(count int) []string {
	var waitGroup sync.WaitGroup
	names := make([]string, count)

	for i := 0; i < count; i++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			names[index] = namegen.Generate()
		}(i)
	}

	waitGroup.Wait()

	return names
}
