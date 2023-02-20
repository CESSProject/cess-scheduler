/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package proof

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"math/big"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/db"
)

type RSAKeyPair struct {
	Spk *rsa.PublicKey
	Ssk *rsa.PrivateKey
}

var key *RSAKeyPair

func init() {
	key = &RSAKeyPair{
		Spk: new(rsa.PublicKey),
		Ssk: new(rsa.PrivateKey),
	}
}

func KeyGen() RSAKeyPair {
	ssk, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	return RSAKeyPair{
		Spk: &ssk.PublicKey,
		Ssk: ssk,
	}
}

func GetKey(d db.Cacher) (*RSAKeyPair, error) {
	if key.Spk.N == nil || key.Spk.E == 0 {
		SetKey(d)
	}
	if key.Spk.N == nil || key.Spk.E == 0 {
		return nil, errors.New("key is nil")
	}
	return key, nil
}

func SetKey(cace db.Cacher) error {
	if cace == nil {
		return errors.New("cace is nil")
	}
	if key.Spk.N == nil || key.Spk.E == 0 {
		val, err := cace.Get([]byte(configs.SigKey_E))
		if err != nil {
			return err
		}
		E_bigint, ok := new(big.Int).SetString(string(val), 10)
		if !ok {
			return errors.New("Set string to E")
		}
		val, err = cace.Get([]byte(configs.SigKey_N))
		if err != nil {
			return err
		}
		key.Spk.N, ok = new(big.Int).SetString(string(val), 10)
		if !ok {
			return errors.New("Set string to N")
		}
		key.Spk.E = int(E_bigint.Int64())
	}
	return nil
}
