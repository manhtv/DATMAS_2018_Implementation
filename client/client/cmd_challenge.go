package client

import (
	"github.com/spf13/cobra"
	"log"
	"strings"
	"strconv"
	"github.com/racin/DATMAS_2018_Implementation/crypto"
	conf "github.com/racin/DATMAS_2018_Implementation/configuration"
	"crypto/rand"
	"math/big"
	"encoding/json"
	tmtypes "github.com/tendermint/tendermint/types"
	"time"
	"fmt"
	"github.com/racin/DATMAS_2018_Implementation/types"
)

var challengeCmd = &cobra.Command{
	Use:     "challenge [CID] [challenge] [proof]",
	Aliases: []string{"challenge"},
	Short:   "Challenge storage nodes",
	Long:    `Challenge storage nodes to prove that they still possess all the data for a CID.`,
	Run: func(cmd *cobra.Command, args []string) {
		var challengeIndices []uint64
		var proof string
		if len(args) < 1 {
			log.Fatal("Not enough arguments.")
		} else if len(args) == 3 {
			proof = args[2]
			strArr := strings.Split(args[1], ",")
			for i, val := range strArr{
				if index, err := strconv.Atoi(val); err == nil {
					challengeIndices[i] = uint64(index)
				}
			}
		}
		cid := args[0]
		var challenge *crypto.SignedStruct
		var hashChal string
		if challengeIndices == nil{
			me := types.GetMetadata(cid)
			if me == nil {
				log.Fatal("Could not find stored metadata for CID: " + cid)
			}
			challenge, hashChal, proof = me.StorageSample.GenerateChallenge(TheClient.privKey)
		} else {
			nonce, err := rand.Int(rand.Reader, new(big.Int).SetUint64(2 << 52)) // 9007199254740992
			if err != nil {
				log.Fatal(err.Error()) // Could not generate nonce.
			}
			chal := &crypto.StorageChallenge{Identity:TheClient.fingerprint, Cid:cid, Challenge:challengeIndices, Nonce:float64(nonce.Int64())}
			if challenge, err = crypto.SignStruct(chal, TheClient.privKey); err != nil {
				log.Fatal(err.Error())
			}
			hashChal = crypto.HashStruct(challenge)
		}
		byteArr, err := json.Marshal(challenge);
		if err != nil {
			log.Fatal(err.Error())
		}

		queryResult, err := TheClient.TMClient.ABCIQuery("/challenge", byteArr)
		if err != nil {
			log.Fatal(err.Error())
		}
		if queryResult.Response.Code != uint32(types.CodeType_OK) {
			log.Fatal(queryResult.Response.Log)
		}
		newBlockCh := make(chan interface{}, 1)
		if err := TheClient.SubToNewBlock(newBlockCh); err != nil {
			log.Fatal("Could not subscribe to new block events. Error: ", err.Error())
		}
		foundChallenge := false
		for {
			select {
			case b := <-newBlockCh:
				evt := b.(tmtypes.EventDataNewBlock)
				// Validate
				if err := evt.Block.ValidateBasic(); err != nil {
					// System is broken. Notify administrators
					log.Fatal("Could not validate latest block. Error: ", err.Error())
				}
				for i := int64(0); i < evt.Block.NumTxs; i++ {
					if _, tx, err := types.UnmarshalTransaction([]byte(evt.Block.Txs[i])); err == nil {
						// Is this an array of SignedStruct (Base type StorageChallengeProof).
						signedStructArr, ok := tx.Data.([]crypto.SignedStruct);
						if !ok {
							continue
						}
						fmt.Printf("-------------\nBlock height: %v\n", evt.Block.Height)
						fmt.Printf("-------------\nTransaction ID:: %v / %v\n", i, evt.Block.NumTxs)
						for _, signedStruct := range signedStructArr {
							scp := signedStruct.Base.(*crypto.StorageChallengeProof)
							// A response to the random challenge.
							if hashChal != crypto.HashStruct(scp.Base) {
								fmt.Printf("Node: %v. Random challenge. Got proof: %v\n", scp.Identity, scp.Proof)
								continue
							}
							// A response to our challenge.
							foundChallenge = true
							if proof == scp.Proof {
								fmt.Printf("Node: %v. Proof matched. Got: %v\n", scp.Identity, proof)
							} else {
								fmt.Printf("Node: %v. Proof did not match. Wanted: %v, Got: %v\n", scp.Identity, proof, scp.Proof)
							}
						}

					}
				}
				if foundChallenge {
					return
				}
			case <-time.After(time.Duration(conf.ClientConfig().NewBlockTimeout) * time.Second):
				fmt.Println("Could not verify the ledger within the timeout. The proof may still be published on the ledger.")
				return
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(challengeCmd)
}