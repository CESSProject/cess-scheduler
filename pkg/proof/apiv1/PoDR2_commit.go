package proof

import (
	"cess-scheduler/tools"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"

	"os"

	"github.com/Nik-U/pbc"
)

func (commit PoDR2Commit) PoDR2ProofCommit(ssk []byte, sharedParams string, segmentSize int64) (<-chan PoDR2CommitResponse, error) {
	responseCh := make(chan PoDR2CommitResponse, 1)
	var res PoDR2CommitResponse
	pairing, err := pbc.NewPairingFromString(sharedParams)
	if err != nil {
		return nil, err
	}
	privateKey := pairing.NewZr().SetBytes(ssk)
	file, err := os.Stat(commit.FilePath)
	if err != nil {
		return nil, err
	}
	matrix, n, err := tools.Split(commit.FilePath, commit.BlockSize, file.Size())
	T := FileTagT{}
	T.T0.N = int64(n)
	T.T0.Name = pairing.NewZr().Rand().Bytes()
	U_num := commit.BlockSize / segmentSize
	if commit.BlockSize%segmentSize != 0 {
		U_num++
	}
	T.T0.U = make([][]byte, U_num)

	for i := int64(0); i < U_num; i++ {
		result := pairing.NewG2().Rand().Bytes()
		T.T0.U[i] = result
	}
	tmp, err := json.Marshal(T.T0)
	if err != nil {
		return nil, err
	}

	hashed_t_0 := pairing.NewG2().SetFromStringHash(string(tmp), sha256.New())
	t_0_signature := pairing.NewG2().PowZn(hashed_t_0, privateKey)
	T.Signature = t_0_signature.Bytes()
	res.T = T
	res.Sigmas = make([][]byte, n)
	//g1wait := make(chan struct{}, n)
	for i := int64(0); i < int64(n); i++ {
		//go func(i int64) {
		res.Sigmas[i] = GenerateAuthenticator(i, commit.BlockSize, res.T.T0, matrix[i], privateKey, pairing, segmentSize)
		//g1wait <- struct{}{}
		//}(i)
	}
	// for i := uint64(0); i < n; i++ {
	// 	<-g1wait
	// }
	res.StatueMsg.StatusCode = Success
	res.StatueMsg.Msg = "PoDR2ProofCommit success"
	responseCh <- res
	return responseCh, nil
}

func GenerateAuthenticator(i int64, s int64, T0 T0, piece []byte, Alpha *pbc.Element, pairing *pbc.Pairing, segmentSize int64) []byte {
	productory := pairing.NewG2()
	//for j := int64(0); j < s; j++ {
	//	//mij
	//	piece_sigle := pairing.NewZr().SetFromHash([]byte{piece[j]})
	//	//uj^mij
	//	productory.Mul(productory, pairing.NewG2().PowZn(T0.U[j], piece_sigle))
	//}
	U_num := s / segmentSize
	if s%segmentSize != 0 {
		U_num++
	}
	for j := int64(0); j < U_num; j++ {
		if j == U_num-1 {
			piece_sigle := pairing.NewZr().SetFromHash(piece[j*segmentSize:])
			//uj^mij
			productory.Mul(productory, pairing.NewG2().PowZn(pairing.NewG2().SetBytes(T0.U[j]), piece_sigle))
			continue
		}
		//mij
		piece_sigle := pairing.NewZr().SetFromHash(piece[j*segmentSize : (j+1)*segmentSize])
		//uj^mij
		productory.Mul(productory, pairing.NewG2().PowZn(pairing.NewG2().SetBytes(T0.U[j]), piece_sigle))
	}
	//H(name||i) · uj^mij
	innerProduct := pairing.NewG2().Mul(productory, buildHashNameElement(pairing, T0.Name, i+1))
	return pairing.NewG2().PowZn(innerProduct, Alpha).Bytes()
}

func buildHashNameElement(pairing *pbc.Pairing, t0Name []byte, i int64) *pbc.Element {
	indexBytes := make([]byte, 4)
	binary.PutVarint(indexBytes, i)
	hash_array := sha256.Sum256(append(pairing.NewZr().SetBytes(t0Name).Bytes(), indexBytes...))
	return pairing.NewG2().SetFromHash(hash_array[:])
}
