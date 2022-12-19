// Copyright 2019 DeepMap, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run github.com/four-fingers/oapi-codegen/cmd/oapi-codegen --config=types.cfg.yaml ../../petstore-expanded.yaml
//go:generate go run github.com/four-fingers/oapi-codegen/cmd/oapi-codegen --config=server.cfg.yaml ../../petstore-expanded.yaml

package api

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gofiber/fiber/v2"
)

type PetStore struct {
	Pets   map[int64]Pet
	NextId int64
	Lock   sync.Mutex
}

func NewPetStore() *PetStore {
	return &PetStore{
		Pets:   make(map[int64]Pet),
		NextId: 1000,
	}
}

// This function wraps sending of an error in the Error format, and
// handling the failure to marshal that.
func sendPetstoreError(c *fiber.Ctx, code int, message string) error {
	petErr := Error{
		Code:    int32(code),
		Message: message,
	}
	return c.Status(code).JSON(petErr)
}

// Here, we implement all of the handlers in the ServerInterface
func (p *PetStore) FindPets(c *fiber.Ctx, params FindPetsParams) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	var result []Pet

	for _, pet := range p.Pets {
		if params.Tags != nil {
			// If we have tags,  filter pets by tag
			for _, t := range *params.Tags {
				if pet.Tag != nil && (*pet.Tag == t) {
					result = append(result, pet)
				}
			}
		} else {
			// Add all pets if we're not filtering
			result = append(result, pet)
		}

		if params.Limit != nil {
			l := int(*params.Limit)
			if len(result) >= l {
				// We're at the limit
				break
			}
		}
	}
	return c.Status(http.StatusOK).JSON(result)
}

func (p *PetStore) AddPet(c *fiber.Ctx) error {
	// We expect a NewPet object in the request body.
	var newPet NewPet
	err := c.BodyParser(&newPet)
	if err != nil {
		return sendPetstoreError(c, http.StatusBadRequest, "Invalid format for NewPet")
	}
	// We now have a pet, let's add it to our "database".

	// We're always asynchronous, so lock unsafe operations below
	p.Lock.Lock()
	defer p.Lock.Unlock()

	// We handle pets, not NewPets, which have an additional ID field
	var pet Pet
	pet.Name = newPet.Name
	pet.Tag = newPet.Tag
	pet.Id = p.NextId
	p.NextId = p.NextId + 1

	// Insert into map
	p.Pets[pet.Id] = pet

	// Now, we have to return the NewPet
	return c.Status(http.StatusCreated).JSON(pet)
}

func (p *PetStore) FindPetByID(c *fiber.Ctx, petId int64) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	pet, found := p.Pets[petId]
	if !found {
		return sendPetstoreError(c, http.StatusNotFound, fmt.Sprintf("Could not find pet with ID %d", petId))
	}
	return c.Status(http.StatusOK).JSON(pet)
}

func (p *PetStore) DeletePet(c *fiber.Ctx, id int64) error {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	_, found := p.Pets[id]
	if !found {
		return sendPetstoreError(c, http.StatusNotFound, fmt.Sprintf("Could not find pet with ID %d", id))
	}
	delete(p.Pets, id)
	c.Status(http.StatusNoContent)
	return nil
}
