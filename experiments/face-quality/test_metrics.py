import unittest
import numpy as np
from metrics import freeze_threshold, evaluate

class Accounting(unittest.TestCase):
    def test_missing_faces_are_unresolved_not_correct(self):
        records=[(s,i,None) for s in range(11,41) for i in range(1,11)]
        r=evaluate(records,0.5)
        self.assertEqual((r['known_unresolved'],r['unknown_unprocessed'],r['known_correct']),(180,100,0))
        self.assertEqual((r['unscored_same_pairs'],r['unscored_different_pairs']),(1350,43500))
    def test_collapsed_embeddings_force_rejection_threshold(self):
        t,_=freeze_threshold([(1,1,np.array([1.,0.])),(2,1,np.array([1.,0.]))])
        self.assertGreater(t,1.0)
    def test_correct_queries_and_unknown_rejection(self):
        vectors=np.eye(30,dtype=np.float32)
        records=[(s,i,vectors[s-11]) for s in range(11,41) for i in range(1,11)]
        r=evaluate(records,0.5)
        self.assertEqual((r['known_correct'],r['known_wrong'],r['unknown_rejected']),(180,0,100))
        self.assertEqual((r['same_pair_misses'],r['different_pair_accepts']),(0,0))
        # One unknown's nine other photos still reject; this query imitates an enrolled identity.
        records[-1]=(40,10,vectors[0])
        r=evaluate(records,0.5)
        self.assertEqual(r['unknown_false_accept'],1)
    def test_threshold_is_from_calibration_only_and_ambiguous_scores_stay_unknown(self):
        t,m=freeze_threshold([(1,1,np.array([1.,0.])),(2,1,np.array([0.,1.]))])
        self.assertEqual((t,m),(0.363,0.0))
        records=[(s,i,np.array([1.,0.])) for s in range(11,41) for i in range(1,11)]
        r=evaluate(records,t)
        self.assertEqual((r['known_unresolved'],r['unknown_rejected']),(180,100))

if __name__=='__main__': unittest.main()
