"""Aggregate evaluation accounting. Missing faces never become correct known IDs."""
import numpy as np

THRESHOLDS = [0.363, 0.4, 0.45, 0.5, 0.55, 0.6, 0.65, 0.7, 0.75, 0.8, 0.85, 0.9, 1.000001]

def freeze_threshold(records):
    negatives = [float(a[2] @ b[2]) for i,a in enumerate(records) for b in records[i+1:]
                 if a[0] != b[0] and a[2] is not None and b[2] is not None]
    assert negatives, 'no calibration negatives'
    maximum = max(negatives)
    return next((t for t in THRESHOLDS if t >= maximum+0.02), THRESHOLDS[-1]), maximum

def evaluate(records, threshold):
    enrollment = [(s,v) for s,i,v in records if 11 <= s <= 30 and i == 1 and v is not None]
    result = {'known_total':180, 'known_correct':0, 'known_wrong':0, 'known_unresolved':0,
              'unknown_total':100, 'unknown_false_accept':0, 'unknown_rejected':0, 'unknown_unprocessed':0,
              'enrollments_available':len(enrollment), 'scored_same_pairs':0, 'same_pair_misses':0,
              'unscored_same_pairs':0, 'scored_different_pairs':0, 'different_pair_accepts':0,
              'unscored_different_pairs':0}
    for s,i,v in records:
        if s <= 30 and i == 1: continue
        known = s <= 30
        if v is None or not enrollment:
            result['known_unresolved' if known else 'unknown_unprocessed'] += 1
            continue
        scores = sorted([(float(v @ ev), es) for es,ev in enrollment], reverse=True)
        accepted = scores[0][0] >= threshold and (len(scores)==1 or scores[0][0]-scores[1][0] >= 0.05)
        if known:
            key = 'known_unresolved' if not accepted else 'known_correct' if scores[0][1]==s else 'known_wrong'
        else:
            key = 'unknown_false_accept' if accepted else 'unknown_rejected'
        result[key] += 1
    for j,(s,i,v) in enumerate(records):
        for other,_,w in records[j+1:]:
            same = s == other
            label = 'same' if same else 'different'
            if v is None or w is None:
                result['unscored_'+label+'_pairs'] += 1
                continue
            result['scored_'+label+'_pairs'] += 1
            accepted = float(v @ w) >= threshold
            if same and not accepted: result['same_pair_misses'] += 1
            if not same and accepted: result['different_pair_accepts'] += 1
    assert result['known_correct']+result['known_wrong']+result['known_unresolved'] == 180
    assert result['unknown_false_accept']+result['unknown_rejected']+result['unknown_unprocessed'] == 100
    return result
